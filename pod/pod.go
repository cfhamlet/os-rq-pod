package pod

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/go-redis/redis/v7"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/viper"
)

// ResultAndErrorFunc TODO
type ResultAndErrorFunc func() (Result, error)

// Result TODO
type Result map[string]interface{}

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
		Init,
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
	err = pod.setStatus(Preparing)
	pod.Unlock()
	if err != nil {
		return
	}

	err = pod.queueBox.LoadQueues()
	if err == nil {
		err = pod.queueBox.LoadPaused()
	}
	pod.Lock()
	defer pod.Unlock()

	if err == nil {
		err = pod.setStatus(Working)
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
	err = pod.setStatus(Stopping)
	if err == nil {
		err = pod.setStatus(Stopped)
	}
	return
}

// metaInfo TODO
func (pod *Pod) metaInfo() (result Result) {
	return Result{
		"queues_info": pod.queueBox.Info(),
		"stats":       pod.stats.Stats(),
		"status":      pod.status,
		"process": Result{
			"pid":    os.Getpid(),
			"memory": utils.MemoryInfo(pod.Process),
			"cpu": Result{
				"percent": utils.CPUPercent(pod.Process),
			},
		},
	}
}

// Info TODO
func (pod *Pod) Info() (result Result, err error) {
	pod.RLock()
	defer pod.RUnlock()

	result = pod.metaInfo()
	t := time.Now()
	memoryInfo, err := pod.Client.Info("memory").Result()
	r := Result{"_cost_ms_": utils.SinceMS(t)}

	if err == nil {
		k, v := utils.ParseRedisInfo(memoryInfo, "used_memory_rss")
		if k != "" {
			r[k] = v
		}
	} else {
		err = fmt.Errorf("redis error %w", err)
	}
	result["redis"] = r

	return
}

// GetRequest TODO
func (pod *Pod) GetRequest(qid QueueID) (req *request.Request, err error) {
	pod.RLock()
	defer pod.RUnlock()

	if pod.status != Working {
		err = UnavailableError(pod.status)
	} else {
		req, err = pod.queueBox.GetRequest(qid)
	}
	return
}

// AddRequest TODO
func (pod *Pod) AddRequest(rawReq *request.RawRequest) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (result Result, err error) {
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
		},
	)
}

func (pod *Pod) withLockRLockOnWorkStatus(f ResultAndErrorFunc, lock bool) (result Result, err error) {
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

func (pod *Pod) withRLockOnWorkStatus(f ResultAndErrorFunc) (Result, error) {
	return pod.withLockRLockOnWorkStatus(f, false)
}

func (pod *Pod) withLockOnWorkStatus(f ResultAndErrorFunc) (Result, error) {
	return pod.withLockRLockOnWorkStatus(f, true)
}

// PauseQueue TODO
func (pod *Pod) PauseQueue(qid QueueID) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.SetStatus(qid, QueuePaused)
		},
	)
}

// ResumeQueue TODO
func (pod *Pod) ResumeQueue(qid QueueID) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.SetStatus(qid, QueueWorking)
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
func (pod *Pod) ForceSyncQueue(qid QueueID) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.SyncQueue(qid, true)
		},
	)
}

// SyncQueue TODO
func (pod *Pod) SyncQueue(qid QueueID) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.SyncQueue(qid, false)
		},
	)
}

// QueueInfo TODO
func (pod *Pod) QueueInfo(qid QueueID) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.QueueInfo(qid)
		},
	)
}

// Pause TODO
func (pod *Pod) Pause() (Result, error) {
	return pod.withLockOnWorkStatus(
		func() (result Result, err error) {
			err = pod.setStatus(Paused)
			if err == nil {
				result = pod.metaInfo()
			}
			return
		},
	)
}

// Resume TODO
func (pod *Pod) Resume() (Result, error) {
	return pod.withLockOnWorkStatus(
		func() (result Result, err error) {
			err = pod.setStatus(Working)
			if err == nil {
				result = pod.metaInfo()
			}
			return
		},
	)
}

// ViewQueue TODO
func (pod *Pod) ViewQueue(qid QueueID, start int64, end int64) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (Result, error) {
			return pod.queueBox.ViewQueue(qid, start, end)
		},
	)
}

// ViewQueues TODO
func (pod *Pod) ViewQueues(k int, start int, status QueueStatus) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (result Result, err error) {
			result = pod.queueBox.ViewQueues(k, start, status)
			return
		},
	)
}

// Queues TODO
func (pod *Pod) Queues(k int) (Result, error) {
	return pod.withRLockOnWorkStatus(
		func() (result Result, err error) {
			if pod.status != Working {
				err = UnavailableError(pod.status)
			} else {
				result = pod.queueBox.Queues(k)
			}
			return
		},
	)
}

func (pod *Pod) setStatus(status Status) (err error) {
	if pod.status == status {
		return
	}
	e := UnavailableError(pod.status)
	switch pod.status {
	case Init:
		switch status {
		case Preparing:
		default:
			err = e
		}
	case Preparing:
		switch status {
		case Working:
			fallthrough
		case Paused:
		default:
			err = e
		}
	case Working:
		switch status {
		case Paused:
			fallthrough
		case Stopping:
		default:
			err = e
		}
	case Paused:
		switch status {
		case Working:
			fallthrough
		case Stopping:
		default:
			err = e
		}
	case Stopping:
		switch status {
		case Stopped:
		default:
			err = e
		}
	case Stopped:
		err = e
	}
	if err == nil {
		pod.status = status
	}
	return
}
