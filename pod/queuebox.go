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
	queueMap       map[QueueID]*Queue
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
			_, err = box.addQueue(qid)
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
			len(box.queueMap), box.pod.stats.RequestNum())
		if cursor == 0 {
			break
		}
	}
	log.Logger.Debugf("load queues finish, queues %d, requests %d",
		len(box.queueMap), box.pod.stats.RequestNum())

	return

}

// DeleteIdleQueue TODO
func (box *QueueBox) DeleteIdleQueue(qid QueueID) (err error) {
	box.Lock()
	defer box.Unlock()
	queue, ok := box.queueMap[qid]
	if ok && queue.Idle() {
		err = box.deleteQueue(qid)
	} else {
		err = QueueNotExist
	}
	return
}

// SyncQueue TODO
func (box *QueueBox) SyncQueue(qid QueueID, force bool) (result Result, err error) {
	box.RLock()

	queue, ok := box.queueMap[qid]
	if !ok {
		if force {
			queue, err = box.addQueue(qid)
		} else {
			err = QueueNotExist
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
	queue, ok := box.queueMap[qid]
	if !ok {
		queue, err = box.addQueue(qid)
	}

	if err == nil {
		result, err = queue.Put(req)
		box.RUnlock()
		if err != nil {
			switch err.(type) {
			case UnavailableError:
			default:
				_ = box.DeleteIdleQueue(qid)
			}
		}
	} else {
		box.RUnlock()
	}
	return
}

// GetRequest TODO
func (box *QueueBox) GetRequest(qid QueueID) (result Result, err error) {
	box.RLock()
	queue, ok := box.queueMap[qid]
	if ok {
		var qsize int64
		result, qsize, err = queue.Get()
		box.RUnlock()
		if (err == nil && qsize <= 0) || err == redis.Nil {
			_, _ = box.SyncQueue(qid, false)
			if err == redis.Nil {
				err = QueueNotExist
			}
		}
	} else {
		box.RUnlock()
		err = QueueNotExist
	}
	return
}

func (box *QueueBox) addQueue(qid QueueID) (queue *Queue, err error) {
	box.cLocker.Lock()
	defer box.cLocker.Unlock()

	queue, ok := box.queueMap[qid]
	if ok {
		return
	}

	queue = NewQueue(box.pod, qid, QueueUndefined)
	_, err = queue.SetStatus(QueueWorking)
	if err != nil {
		queue = nil
		return
	}
	box.queueMap[qid] = queue
	box.statusQueueIDs[QueueWorking].Add(qid)
	return
}

func (box *QueueBox) deleteQueue(qid QueueID) (err error) {
	queue, ok := box.queueMap[qid]
	if !ok {
		err = QueueNotExist
		return
	}
	status := queue.Status()
	delete(box.queueMap, qid)
	box.statusQueueIDs[status].Delete(qid)

	return
}

// DeleteQueue TODO
func (box *QueueBox) DeleteQueue(qid QueueID) (result Result, err error) {
	box.Lock()
	defer box.Unlock()
	result, err = box.clearQueue(qid)
	if err != nil {
		err = box.deleteQueue(qid)
	}
	return
}

func (box *QueueBox) clearQueue(qid QueueID) (result Result, err error) {
	queue, ok := box.queueMap[qid]
	if !ok {
		err = QueueNotExist
		return
	}
	return queue.Clear()
}

// ViewQueue TODO
func (box *QueueBox) ViewQueue(qid QueueID, start int64, end int64) (result Result, err error) {
	box.RLock()
	defer box.RUnlock()
	return nil, nil
}

// ClearQueue TODO
func (box *QueueBox) ClearQueue(qid QueueID) (result Result, err error) {
	box.RLock()
	defer box.RUnlock()
	return box.clearQueue(qid)
}

// UpdateQueueStatus TODO
func (box *QueueBox) UpdateQueueStatus(qid QueueID, status QueueStatus) (result Result, err error) {
	box.Lock()
	defer box.Unlock()

	queue, ok := box.queueMap[qid]

	if !ok {
		err = QueueNotExist
		return
	}

	oldStatus := queue.Status()
	if oldStatus == status {
		result, err = queue.Info()
		return
	}

	result, err = queue.SetStatus(status)

	if err != nil {
		return
	}
	box.statusQueueIDs[oldStatus].Delete(qid)
	box.statusQueueIDs[status].Add(qid)
	return
}

// QueuesNum TODO
func (box *QueueBox) QueuesNum(status QueueStatus) int {
	box.RLock()
	defer box.RUnlock()
	if status == QueueUndefined {
		return len(box.queueMap)
	}
	return box.statusQueueIDs[status].Size()
}

// Info TODO
func (box *QueueBox) Info() (result Result) {
	box.RLock()
	defer box.RUnlock()
	result = Result{}
	for k, v := range box.statusQueueIDs {
		result[string(k)] = v.Size()
	}
	return
}

// QueueInfo TODO
func (box *QueueBox) QueueInfo(qid QueueID) (result Result, err error) {
	box.RLock()
	defer box.RUnlock()
	queue, ok := box.queueMap[qid]
	if !ok {
		err = QueueNotExist
	}
	return queue.Info()
}

func (box *QueueBox) fillQueues(iter slicemap.Iterator) []Result {
	out := []Result{}
	iter.Iter(
		func(item slicemap.Item) {
			qid := item.(QueueID)
			queue, ok := box.queueMap[qid]
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
