package core

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/go-redis/redis/v7"
	"github.com/prep/average"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/viper"
)

// Status type
type Status string

// Status enum
const (
	Working   Status = "working"
	Paused    Status = "paused"
	Preparing Status = "preparing"
	Stopping  Status = "stopping"
)

// Result TODO
type Result map[string]interface{}

// Stats TODO
type Stats struct {
	requestNum      int64
	requestInputWS  *average.SlidingWindow
	requestOutputWS *average.SlidingWindow
}

// Stats TODO
func (stats *Stats) Stats() Result {
	result := Result{}
	result["request_num"] = stats.RequestNum()
	result["request_input_5s"], _ = stats.requestInputWS.Total(5 * time.Second)
	result["request_output_5s"], _ = stats.requestOutputWS.Total(5 * time.Second)
	return result
}

// IncrRequestNum TODO
func (stats *Stats) IncrRequestNum(d int64) int64 {
	return atomic.AddInt64(&(stats.requestNum), d)
}

// RequestNum TODO
func (stats *Stats) RequestNum() int64 {
	return atomic.LoadInt64(&(stats.requestNum))
}

// UpdateInput TODO
func (stats *Stats) UpdateInput(d int64) {
	stats.IncrRequestNum(d)
	stats.requestInputWS.Add(d)
}

// UpdateOutput TODO
func (stats *Stats) UpdateOutput(d int64) {
	stats.IncrRequestNum(0 - d)
	stats.requestOutputWS.Add(d)
}

// NewStats TODO
func NewStats() *Stats {
	return &Stats{0,
		average.MustNew(time.Minute, time.Second),
		average.MustNew(time.Minute, time.Second),
	}
}

// Pod TODO
type Pod struct {
	Client  *redis.Client
	Process *process.Process

	conf     *viper.Viper
	queueBox *QueueBox
	stats    *Stats
	status   Status
	stLocker *sync.RWMutex
	qLocker  *sync.RWMutex
}

// NewPod creates RQ object
func NewPod(conf *viper.Viper, client *redis.Client) (pod *Pod, err error) {
	proc, err := utils.NewProcess()
	if err != nil {
		return
	}
	stats := NewStats()

	pod = &Pod{
		client, proc, conf, nil, stats, Preparing,
		&sync.RWMutex{}, &sync.RWMutex{},
	}

	queueBox := NewQueueBox(pod)
	pod.queueBox = queueBox
	return
}

func loadQueues(pod *Pod, keys []string) (err error) {
	for _, key := range keys {
		qid, e := QueueIDFromRedisKey(key)
		if e != nil {
			log.Logger.Warning("invalid key", key, e)
		} else {
			_, err = pod.queueBox.AddQueue(qid)
			if err != nil {
				break
			}
		}
	}
	return
}

// LoadQueues TODO
func LoadQueues(pod *Pod) (err error) {

	log.Logger.Debug("load queues start")
	var cursor uint64
	for {
		var keys []string
		keys, cursor, err = pod.Client.Scan(cursor, RedisQueueKeyPrefix+"*", 2000).Result()
		if err == nil {
			err = loadQueues(pod, keys)
		}
		if err != nil {
			return
		}
		log.Logger.Debugf("loading queues, queues %d, requests %d",
			pod.queueBox.QueueNum(QueueNilStatus), pod.stats.RequestNum())
		if cursor == 0 {
			break
		}
	}
	log.Logger.Debugf("load queues finish, queues %d, requests %d",
		pod.queueBox.QueueNum(QueueNilStatus), pod.stats.RequestNum())

	_, err = pod.setStatus(Working)
	return
}

// metaInfo TODO
func (pod *Pod) metaInfo() (result Result) {

	result = Result{}
	result["queues"] = pod.queueBox.Info()
	result["stats"] = pod.stats.Stats()
	result["status"] = pod.status

	v, _ := pod.Process.MemoryInfo()
	c, _ := pod.Process.CPUPercent()
	result["process"] = Result{"memory": v, "cpu": Result{"percent": c}}

	return
}

// Conf TODO
func (pod *Pod) Conf() *viper.Viper {
	return pod.conf
}

// Info TODO
func (pod *Pod) Info() (result Result, err error) {
	pod.stLocker.RLock()
	defer pod.stLocker.RUnlock()

	pod.qLocker.Lock()
	result = pod.metaInfo()
	pod.qLocker.Unlock()

	t := time.Now()
	memoryInfo, err := pod.Client.Info("memory").Result()

	if err == nil {
		r := Result{"latency_ms": float64(time.Since(t)) / 1000000}
		k, v := utils.ParseRedisInfo(memoryInfo, "used_memory_rss")
		if k != "" {
			r[k] = v
		}
		result["redis"] = r
	} else {
		err = fmt.Errorf("redis error %w", err)
	}

	return
}

// GetRequest TODO
func (pod *Pod) GetRequest(qid QueueID) (result Result, err error) {
	pod.stLocker.RLock()
	defer pod.stLocker.RUnlock()

	if pod.status != Working {
		err = UnavailableError(fmt.Sprintf("pod %s", pod.status))
		return
	}

	pod.qLocker.RLock()
	queue, ok := pod.queueBox.GetQueue(qid)
	if ok {
		result, err = queue.Get()
		qsize := queue.QueueSize()
		pod.qLocker.RUnlock()
		if (err == nil && qsize <= 0) || err == redis.Nil {
			_, _ = pod.SyncQueue(qid)
			if err == redis.Nil {
				err = QueueNotExist
			}
		}
	} else {
		pod.qLocker.RUnlock()
		err = QueueNotExist
	}
	return
}

// AddRequest TODO
func (pod *Pod) AddRequest(rawReq *request.RawRequest) (result Result, err error) {
	pod.stLocker.RLock()
	defer pod.stLocker.RUnlock()

	if pod.status != Working {
		err = UnavailableError(fmt.Sprintf("pod %s", pod.status))
		return
	}

	var req *request.Request
	req, err = request.NewRequest(rawReq)
	if err == nil {
		qid := QueueIDFromRequest(req)
		pod.qLocker.RLock()
		queue, ok := pod.queueBox.GetQueue(qid)
		if !ok {
			queue, err = pod.queueBox.AddQueue(qid)
		}
		if err == nil {
			result, err = queue.Put(req)
			pod.qLocker.RUnlock()
			if err != nil {
				switch err.(type) {
				case UnavailableError:
				default:
					_ = pod.dropIdleQueue(qid)
				}
			}
		} else {
			pod.qLocker.RUnlock()
		}
	}

	return
}

func (pod *Pod) operateQueue(qid QueueID, optName string, rlock bool) (result Result, err error) {

	if rlock {
		pod.qLocker.RLock()
		defer pod.qLocker.RUnlock()
	}

	queue, ok := pod.queueBox.GetQueue(qid)

	if ok {
		results := reflect.ValueOf(queue).MethodByName(optName).Call([]reflect.Value{})
		result = results[0].Interface().(Result)
		e := results[1].Interface()
		if e != nil {
			err = e.(error)
		}
	} else {
		err = QueueNotExist
	}

	return
}

// PauseQueue TODO
func (pod *Pod) PauseQueue(qid QueueID) (result Result, err error) {
	return pod.queueBox.UpdateQueueStatus(qid, QueuePaused)
}

// ResumeQueue TODO
func (pod *Pod) ResumeQueue(qid QueueID) (result Result, err error) {
	return pod.queueBox.UpdateQueueStatus(qid, QueueWorking)
}

// DeleteQueue TODO
func (pod *Pod) DeleteQueue(qid QueueID) (Result, error) {
	return pod.clearQueue(qid, true)
}

// ClearQueue TODO
func (pod *Pod) ClearQueue(qid QueueID) (Result, error) {
	return pod.clearQueue(qid, false)
}

func (pod *Pod) clearQueue(qid QueueID, delete bool) (result Result, err error) {

	if delete {
		pod.qLocker.Lock()
		defer pod.qLocker.Unlock()
	}

	result, err = pod.operateQueue(qid, "Clear", !delete)

	if err == nil && delete {
		err = pod.queueBox.RemoveQueue(qid)
	}

	return
}

// ForceSyncQueue TODO
func (pod *Pod) ForceSyncQueue(qid QueueID) (result Result, err error) {
	return pod.syncQueue(qid, true)
}

// SyncQueue TODO
func (pod *Pod) SyncQueue(qid QueueID) (result Result, err error) {
	return pod.syncQueue(qid, false)
}

// SyncQueue TODO
func (pod *Pod) syncQueue(qid QueueID, force bool) (result Result, err error) {
	pod.qLocker.RLock()

	queue, ok := pod.queueBox.GetQueue(qid)
	if !ok {
		if force {
			queue, err = pod.queueBox.AddQueue(qid)
		} else {
			err = QueueNotExist
			pod.qLocker.RUnlock()
			return
		}
	}

	if err == nil {
		result, err = queue.Sync(true)
		pod.qLocker.RUnlock()

		if err == nil || !ok {
			_ = pod.dropIdleQueue(qid)
		}
	} else {
		pod.qLocker.RUnlock()
	}

	return
}

func (pod *Pod) dropIdleQueue(qid QueueID) (err error) {
	pod.qLocker.Lock()
	defer pod.qLocker.Unlock()

	queue, ok := pod.queueBox.GetQueue(qid)
	if ok {
		if queue.Idle() {
			err = pod.queueBox.RemoveQueue(qid)
		}
	}
	return
}

// QueueInfo TODO
func (pod *Pod) QueueInfo(qid QueueID) (Result, error) {
	return pod.operateQueue(qid, "Info", true)
}

// Pause TODO
func (pod *Pod) Pause() (Result, error) {
	return pod.setStatus(Paused)
}

// Resume TODO
func (pod *Pod) Resume() (Result, error) {
	return pod.setStatus(Working)
}

func (pod *Pod) setStatus(status Status) (Result, error) {
	pod.stLocker.Lock()
	defer pod.stLocker.Unlock()

	pod.status = status
	return pod.metaInfo(), nil

}

// ViewQueue TODO
func (pod *Pod) ViewQueue(qid QueueID, start int64, end int64) (result Result, err error) {
	pod.qLocker.RLock()
	defer pod.qLocker.RUnlock()

	queue, ok := pod.queueBox.GetQueue(qid)
	if ok {
		result, err = queue.View(start, end)
	} else {
		err = QueueNotExist
	}

	return
}

// OrderedQueues TODO
func (pod *Pod) OrderedQueues(k int64, start int64, status QueueStatus) Result {
	return pod.queueBox.OrderedQueues(k, start, status)
}

// RandomQueues TODO
func (pod *Pod) RandomQueues(k int64, status QueueStatus) Result {
	return pod.queueBox.RandomQueues(k, status)
}
