package pod

import (
	"fmt"
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
	Stopped   Status = "stopped"
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
	limiter  *Limiter
}

// NewPod creates RQ object
func NewPod(conf *viper.Viper, client *redis.Client) (pod *Pod) {
	proc, err := utils.NewProcess()
	if err != nil {
		panic(err)
	}
	stats := NewStats()

	pod = &Pod{
		client, proc, conf, nil, stats, Stopped,
		&sync.RWMutex{}, &sync.RWMutex{}, nil,
	}

	queueBox := NewQueueBox(pod)
	pod.queueBox = queueBox

	limiter := NewLimiter(pod)
	pod.limiter = limiter
	return
}

func stopStatus(status Status) bool {
	return status == Stopped || status == Stopping
}

// OnStart TODO
func (pod *Pod) OnStart() (err error) {
	pod.stLocker.Lock()
	if pod.status != Stopped {
		pod.stLocker.Unlock()
		return UnavailableError(pod.status)
	}
	pod.setStatus(Preparing)
	pod.stLocker.Unlock()

	err = pod.queueBox.LoadQueues()
	if err == nil {
		err = pod.start()
	}
	switch err.(type) {
	case UnavailableError:
		if stopStatus(pod.status) {
			log.Logger.Warning("stop when starting")
			return nil
		}
	}
	return
}

// OnStop TODO
func (pod *Pod) OnStop() (err error) {
	pod.stLocker.Lock()
	defer pod.stLocker.Unlock()
	pod.setStatus(Stopping)
	// TODO
	pod.setStatus(Stopped)
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

	result = pod.metaInfo()

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
		err = UnavailableError(pod.status)
		return
	}
	return pod.queueBox.GetRequest(qid)
}

// AddRequest TODO
func (pod *Pod) AddRequest(rawReq *request.RawRequest) (result Result, err error) {
	pod.stLocker.RLock()
	defer pod.stLocker.RUnlock()

	if pod.status != Working {
		err = UnavailableError(pod.status)
		return
	}

	var req *request.Request
	req, err = request.NewRequest(rawReq)
	if err != nil {
		return
	}
	err = pod.limiter.AllowedNewRequest(req)
	if err != nil {
		return
	}
	qid := QueueIDFromRequest(req)

	return pod.queueBox.AddRequest(qid, req)
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
	return pod.queueBox.DeleteQueue(qid)
}

// ClearQueue TODO
func (pod *Pod) ClearQueue(qid QueueID) (Result, error) {
	return pod.queueBox.ClearQueue(qid)
}

// ForceSyncQueue TODO
func (pod *Pod) ForceSyncQueue(qid QueueID) (result Result, err error) {
	return pod.queueBox.SyncQueue(qid, true)
}

// SyncQueue TODO
func (pod *Pod) SyncQueue(qid QueueID) (result Result, err error) {
	return pod.queueBox.SyncQueue(qid, false)
}

// QueueInfo TODO
func (pod *Pod) QueueInfo(qid QueueID) (Result, error) {
	return pod.queueBox.QueueInfo(qid)
}

// Pause TODO
func (pod *Pod) Pause() (result Result, err error) {
	pod.stLocker.Lock()
	defer pod.stLocker.Unlock()

	if pod.status == Paused || pod.status == Working {
		if pod.status == Working {
			pod.setStatus(Paused)
		}
		result = pod.metaInfo()
	} else {
		err = UnavailableError(pod.status)
	}

	return
}

func (pod *Pod) start() (err error) {
	if pod.status == Stopping || pod.status == Stopped {
		err = UnavailableError(pod.status)
	} else if pod.status == Working {
	} else {
		pod.setStatus(Working)
	}
	return
}

// Resume TODO
func (pod *Pod) Resume() (result Result, err error) {
	pod.stLocker.Lock()
	defer pod.stLocker.Unlock()

	if pod.status == Working {
		result = pod.metaInfo()
	} else if pod.status == Paused {
		err = pod.start()
		if err == nil {
			result = pod.metaInfo()
		}
	} else {
		err = UnavailableError(pod.status)
	}

	return
}

func (pod *Pod) setStatus(status Status) {
	pod.status = status
}

// ViewQueue TODO
func (pod *Pod) ViewQueue(qid QueueID, start int64, end int64) (Result, error) {
	return pod.queueBox.ViewQueue(qid, start, end)
}

// ViewQueues TODO
func (pod *Pod) ViewQueues(k int, start int, status QueueStatus) Result {
	return pod.queueBox.ViewQueues(k, start, status)
}

// Queues TODO
func (pod *Pod) Queues(k int) (result Result, err error) {
	pod.stLocker.RLock()
	defer pod.stLocker.RUnlock()
	if pod.status != Working {
		err = UnavailableError(pod.status)
	} else {
		result = pod.queueBox.Queues(k)
	}
	return
}
