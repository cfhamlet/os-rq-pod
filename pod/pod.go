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

// WindowTotal TODO
func WindowTotal(window *average.SlidingWindow, second time.Duration) int64 {
	t, _ := window.Total(second * time.Second)
	return t
}

// Stats TODO
func (stats *Stats) Stats() Result {
	result := Result{}
	result["request_num"] = stats.RequestNum()
	result["request_input_5s"] = WindowTotal(stats.requestInputWS, 5)
	result["request_output_5s"] = WindowTotal(stats.requestOutputWS, 5)
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

// MustNewMinuteWindow TODO
func MustNewMinuteWindow() *average.SlidingWindow {
	return average.MustNew(time.Minute, time.Second)
}

// NewStats TODO
func NewStats() *Stats {
	return &Stats{
		0,
		MustNewMinuteWindow(),
		MustNewMinuteWindow(),
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
	*sync.RWMutex
	limiter *Limiter
}

// NewPod creates RQ object
func NewPod(conf *viper.Viper, client *redis.Client) (pod *Pod) {
	proc, err := utils.NewProcess()
	if err != nil {
		panic(err)
	}
	stats := NewStats()

	pod = &Pod{
		client,
		proc,
		conf,
		nil,
		stats,
		Stopped,
		&sync.RWMutex{},
		nil,
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

func workStatus(status Status) bool {
	return status == Working || status == Paused
}

// OnStart TODO
func (pod *Pod) OnStart() (err error) {
	pod.Lock()
	if pod.status != Stopped {
		pod.Unlock()
		return UnavailableError(pod.status)
	}
	pod.setStatus(Preparing)
	pod.Unlock()

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
	pod.Lock()
	defer pod.Unlock()
	pod.setStatus(Stopping)
	// TODO
	pod.setStatus(Stopped)
	return
}

// MemoryInfo TODO
func MemoryInfo(p *process.Process) *process.MemoryInfoStat {
	m, _ := p.MemoryInfo()
	return m
}

// CPUPercent TODO
func CPUPercent(p *process.Process) float64 {
	c, _ := p.CPUPercent()
	return c
}

// metaInfo TODO
func (pod *Pod) metaInfo() (result Result) {
	return Result{
		"queues": pod.queueBox.Info(),
		"stats":  pod.stats.Stats(),
		"status": pod.status,
		"process": Result{
			"memory": MemoryInfo(pod.Process),
			"cpu": Result{
				"percent": CPUPercent(pod.Process),
			},
		},
	}
}

// Conf TODO
func (pod *Pod) Conf() *viper.Viper {
	return pod.conf
}

// Info TODO
func (pod *Pod) Info() (result Result, err error) {
	pod.RLock()
	defer pod.RUnlock()

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
	pod.RLock()
	defer pod.RUnlock()

	if pod.status != Working {
		err = UnavailableError(pod.status)
		return
	}
	return pod.queueBox.GetRequest(qid)
}

// AddRequest TODO
func (pod *Pod) AddRequest(rawReq *request.RawRequest) (result Result, err error) {
	pod.RLock()
	defer pod.RUnlock()

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

func (pod *Pod) withLockRLockOnWorkStatus(f func() (Result, error), lock bool) (result Result, err error) {
	if lock {
		pod.Lock()
		defer pod.Unlock()
	} else {
		pod.RLock()
		defer pod.RUnlock()
	}
	if !workStatus(pod.status) {
		err = UnavailableError(pod.status)
		return
	}
	return f()
}

func (pod *Pod) withRLockOnWorkStatus(f func() (Result, error)) (result Result, err error) {
	return pod.withLockRLockOnWorkStatus(f, false)
}

func (pod *Pod) withLockOnWorkStatus(f func() (Result, error)) (result Result, err error) {
	return pod.withLockRLockOnWorkStatus(f, true)
}

// PauseQueue TODO
func (pod *Pod) PauseQueue(qid QueueID) (result Result, err error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.UpdateQueueStatus(qid, QueuePaused)
		},
	)
}

// ResumeQueue TODO
func (pod *Pod) ResumeQueue(qid QueueID) (result Result, err error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.UpdateQueueStatus(qid, QueueWorking)
		},
	)
}

// DeleteQueue TODO
func (pod *Pod) DeleteQueue(qid QueueID) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.DeleteQueue(qid)
		},
	)
}

// ClearQueue TODO
func (pod *Pod) ClearQueue(qid QueueID) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.ClearQueue(qid)
		},
	)
}

// ForceSyncQueue TODO
func (pod *Pod) ForceSyncQueue(qid QueueID) (result Result, err error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.SyncQueue(qid, true)
		},
	)
}

// SyncQueue TODO
func (pod *Pod) SyncQueue(qid QueueID) (result Result, err error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.SyncQueue(qid, false)
		},
	)
}

// QueueInfo TODO
func (pod *Pod) QueueInfo(qid QueueID) (result Result, err error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.QueueInfo(qid)
		},
	)
}

func (pod *Pod) start() (err error) {
	if stopStatus(pod.status) {
		err = UnavailableError(pod.status)
	} else if pod.status == Working {
	} else {
		pod.setStatus(Working)
	}
	return
}

// Pause TODO
func (pod *Pod) Pause() (result Result, err error) {
	return pod.withLockOnWorkStatus(
		func() (Result, error) {
			if pod.status == Working {
				pod.setStatus(Paused)
			}
			return pod.metaInfo(), nil
		},
	)
}

// Resume TODO
func (pod *Pod) Resume() (result Result, err error) {
	return pod.withLockOnWorkStatus(
		func() (Result, error) {
			if pod.status == Paused {
				pod.setStatus(Working)
			}
			return pod.metaInfo(), nil
		},
	)
}

func (pod *Pod) setStatus(status Status) {
	pod.status = status
}

// ViewQueue TODO
func (pod *Pod) ViewQueue(qid QueueID, start int64, end int64) (result Result, err error) {
	pod.RLock()
	defer pod.RUnlock()
	if !workStatus(pod.status) {
		err = UnavailableError(pod.status)
		return
	}
	return pod.queueBox.ViewQueue(qid, start, end)
}

// ViewQueues TODO
func (pod *Pod) ViewQueues(k int, start int, status QueueStatus) (result Result, err error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.ViewQueues(k, start, status), nil
		},
	)
}

// Queues TODO
func (pod *Pod) Queues(k int) (result Result, err error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.Queues(k), nil
		},
	)
}
