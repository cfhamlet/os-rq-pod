package pod

import (
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/slicemap"
	"github.com/go-redis/redis/v7"
)

// QueueBox TODO
type QueueBox struct {
	pod            *Pod
	queues         map[QueueID]*Queue
	statusQueueIDs map[QueueStatus]*slicemap.Map
	*sync.RWMutex
	cLocker *sync.Mutex
}

// NewQueueBox TODO
func NewQueueBox(pod *Pod) *QueueBox {
	statusQueueIDs := map[QueueStatus]*slicemap.Map{}
	for _, status := range QueueStatusList {
		statusQueueIDs[status] = slicemap.New()
	}
	return &QueueBox{
		pod,
		map[QueueID]*Queue{},
		statusQueueIDs,
		&sync.RWMutex{},
		&sync.Mutex{},
	}
}

func (box *QueueBox) loadQueues(keys []string) (err error) {
	for _, key := range keys {
		if box.pod.status != Preparing {
			return UnavailableError(box.pod.status)
		}
		qid, e := QueueIDFromRedisKey(key)
		if e != nil {
			log.Logger.Warning("invalid key", key, e)
		} else {
			_, err = box.addQueue(qid, QueueWorking)
			if err != nil {
				break
			}
		}
	}
	return
}

// LoadQueues TODO
func (box *QueueBox) LoadQueues() (err error) {
	box.Lock()
	defer box.Unlock()

	log.Logger.Debug("load queues start")
	var cursor uint64
	for {
		if box.pod.status != Preparing {
			return UnavailableError(box.pod.status)
		}
		var keys []string
		keys, cursor, err = box.pod.Client.Scan(cursor, RedisQueueKeyPrefix+"*", 2000).Result()
		if err == nil {
			err = box.loadQueues(keys)
		}
		if err != nil {
			return
		}
		log.Logger.Debugf("loading queues, queues %d, requests %d",
			len(box.queues), box.pod.stats.RequestNum())
		if cursor == 0 {
			break
		}
	}
	log.Logger.Debugf("load queues finish, queues %d, requests %d",
		len(box.queues), box.pod.stats.RequestNum())

	return
}

// CallByQueue TODO
type CallByQueue func(*Queue) (Result, error)

func (box *QueueBox) withRLock(qid QueueID, f CallByQueue) (result Result, err error) {
	box.RLock()
	defer box.RUnlock()
	return box.mustExist(qid, f)
}

func (box *QueueBox) withLock(qid QueueID, f CallByQueue) (result Result, err error) {
	box.Lock()
	defer box.Unlock()
	return box.mustExist(qid, f)
}

func (box *QueueBox) mustExist(qid QueueID, f CallByQueue) (Result, error) {
	queue, ok := box.queues[qid]
	if !ok {
		return nil, QueueNotExistError(qid.String())
	}
	return f(queue)
}

// DeleteIdleQueue TODO
func (box *QueueBox) DeleteIdleQueue(qid QueueID) (err error) {
	_, err = box.withLock(qid,
		func(queue *Queue) (result Result, err error) {
			if queue.Idle() {
				err = queue.SetStatus(QueueRemoved)
			}
			return
		},
	)
	return err
}

// SyncQueue TODO
func (box *QueueBox) SyncQueue(qid QueueID, force bool) (result Result, err error) {
	box.RLock()

	queue, ok := box.queues[qid]
	if !ok {
		if force {
			queue, err = box.addQueue(qid, QueueInit)
		} else {
			err = QueueNotExistError(qid.String())
			box.RUnlock()
			return
		}
	}

	if err == nil {
		result, err = queue.Sync()
		box.RUnlock()

		if err == nil || !ok {
			_ = box.DeleteIdleQueue(qid)
		}
	} else {
		box.RUnlock()
	}

	return
}

// AddRequest TODO
func (box *QueueBox) AddRequest(qid QueueID, req *request.Request) (result Result, err error) {
	box.RLock()
	queue, ok := box.queues[qid]
	if !ok {
		queue, err = box.addQueue(qid, QueueWorking)
	}

	if err != nil {
		box.RUnlock()
		return
	}

	result, err = queue.Put(req)
	box.RUnlock()
	if err != nil {
		switch err.(type) {
		case UnavailableError:
		default:
			_ = box.DeleteIdleQueue(qid)
		}
	}
	return
}

// GetRequest TODO
func (box *QueueBox) GetRequest(qid QueueID) (result Result, err error) {
	box.RLock()
	queue, ok := box.queues[qid]

	if !ok {
		err = QueueNotExistError(qid.String())
		box.RUnlock()
		return
	}

	var qsize int64
	result, qsize, err = queue.Get()
	box.RUnlock()
	if (err == nil && qsize <= 0) || err == redis.Nil {
		_, _ = box.SyncQueue(qid, false)
		if err == redis.Nil {
			err = QueueNotExistError(qid.String())
		}
	}
	return
}

func (box *QueueBox) addQueue(qid QueueID, status QueueStatus) (queue *Queue, err error) {
	box.cLocker.Lock()
	defer box.cLocker.Unlock()

	queue, ok := box.queues[qid]
	if ok {
		return
	}
	queue = NewQueue(box.pod, qid, QueueInit)
	err = queue.SetStatus(status)
	if err == nil {
		_, err = queue.Sync()
	}
	if err != nil {
		_ = queue.SetStatus(QueueRemoved)
		queue = nil
	}
	return
}

// DeleteQueue TODO
func (box *QueueBox) DeleteQueue(qid QueueID) (Result, error) {
	return box.withLock(qid,
		func(queue *Queue) (result Result, err error) {
			result, err = queue.Clear()
			if err == nil {
				err = queue.SetStatus(QueueRemoved)
			}
			return
		},
	)
}

// ViewQueue TODO
func (box *QueueBox) ViewQueue(qid QueueID, start int64, end int64) (result Result, err error) {
	return box.withRLock(qid,
		func(queue *Queue) (Result, error) {
			return queue.View(start, end)
		},
	)
}

// ClearQueue TODO
func (box *QueueBox) ClearQueue(qid QueueID) (Result, error) {
	return box.withRLock(qid,
		func(queue *Queue) (Result, error) {
			return queue.Clear()
		},
	)
}

// SetStatus TODO
func (box *QueueBox) SetStatus(qid QueueID, status QueueStatus) (Result, error) {
	return box.withLock(qid,
		func(queue *Queue) (result Result, err error) {
			err = queue.SetStatus(status)
			if err == nil {
				result = queue.Info()
			}
			return
		},
	)
}

// QueueInfo TODO
func (box *QueueBox) QueueInfo(qid QueueID) (Result, error) {
	return box.withRLock(qid,
		func(queue *Queue) (result Result, err error) {
			result = queue.Info()
			return
		},
	)
}

func (box *QueueBox) fillQueues(iter slicemap.Iterator) []Result {
	out := []Result{}
	iter.Iter(
		func(item slicemap.Item) {
			qid := item.(QueueID)
			queue, ok := box.queues[qid]
			if ok {
				r := Result{"qid": qid, "qsize": queue.QueueSize()}
				out = append(out, r)
			}
		},
	)
	return out
}

// ViewQueues TODO
func (box *QueueBox) ViewQueues(k int, start int, status QueueStatus) Result {
	box.RLock()
	defer box.RUnlock()
	queueIDs := box.statusQueueIDs[status]
	l := queueIDs.Size()
	var out []Result
	if l <= 0 || k <= 0 {
		out = []Result{}
	} else {
		if start < 0 {
			start = 0
		}
		iterator := slicemap.NewSubIter(queueIDs, start, k)
		out = box.fillQueues(iterator)
	}
	return Result{
		"k":      k,
		"start":  start,
		"queues": out,
		"total":  l,
		"status": status,
	}
}

// Queues TODO
func (box *QueueBox) Queues(k int) Result {
	box.RLock()
	defer box.RUnlock()
	queueIDs := box.statusQueueIDs[QueueWorking]
	l := queueIDs.Size()
	var out []Result
	if l <= 0 || k <= 0 {
		out = []Result{}
	} else {
		iterator := slicemap.NewRandomKIter(queueIDs, k)
		out = box.fillQueues(iterator)
	}
	return Result{
		"k":      k,
		"queues": out,
		"total":  l,
	}
}

// QueuesNum TODO
func (box *QueueBox) QueuesNum(status QueueStatus) int {
	box.Lock()
	defer box.Unlock()
	if status == QueueInit {
		return len(box.queues)
	}
	return box.statusQueueIDs[status].Size()
}

// Info TODO
func (box *QueueBox) Info() (result Result) {
	box.Lock()
	defer box.Unlock()
	result = Result{}
	for k, v := range box.statusQueueIDs {
		result[string(k)] = v.Size()
	}
	return
}
