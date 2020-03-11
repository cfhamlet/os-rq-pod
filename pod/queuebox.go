package pod

import (
	"math/rand"
	"sync"
	"time"
)

// IdxQueue TODO
type IdxQueue struct {
	Idx int
	*Queue
}

// NewIdxQueue TODO
func NewIdxQueue(pod *Pod, qid QueueID, status QueueStatus, idx int) *IdxQueue {
	return &IdxQueue{idx, NewQueue(pod, qid, status)}
}

// IdxQueueMap TODO
type IdxQueueMap map[QueueID]*IdxQueue

// StatusQueueIDSliceMap TODO
type StatusQueueIDSliceMap map[QueueStatus][]QueueID

// QueueBox TODO
type QueueBox struct {
	pod      *Pod
	queues   IdxQueueMap
	queueIDs StatusQueueIDSliceMap
	locker   *sync.RWMutex
}

// NewQueueBox TODO
func NewQueueBox(pod *Pod) *QueueBox {
	queueIDs := StatusQueueIDSliceMap{}
	for _, status := range QueueStatusSlice {
		queueIDs[status] = []QueueID{}
	}
	return &QueueBox{pod, IdxQueueMap{}, queueIDs, &sync.RWMutex{}}
}

// AddQueue TODO
func (box *QueueBox) AddQueue(qid QueueID) (queue *IdxQueue, err error) {
	box.locker.Lock()
	defer box.locker.Unlock()

	queue, ok := box.queues[qid]
	if ok {
		return
	}

	queueIDs := box.queueIDs[QueueWorking]
	queue = NewIdxQueue(box.pod, qid, QueueNilStatus, len(queueIDs))
	_, err = queue.SetStatus(QueueWorking)
	if err != nil {
		queue = nil
		return
	}
	box.queueIDs[QueueWorking] = append(queueIDs, qid)
	box.queues[qid] = queue
	return
}

// RemoveQueue TODO
func (box *QueueBox) RemoveQueue(qid QueueID) error {
	box.locker.Lock()
	defer box.locker.Unlock()

	queue, ok := box.queues[qid]
	if !ok {
		return QueueNotExist
	}

	status := queue.Status()
	queueIDs := box.queueIDs[status]
	lastIdx := len(queueIDs) - 1
	lastQueueID := queueIDs[lastIdx]
	lastQueue := box.queues[lastQueueID]
	lastQueue.Idx = queue.Idx
	queueIDs[queue.Idx] = lastQueueID
	queueIDs[lastIdx] = QueueID{}
	box.queueIDs[status] = queueIDs[:lastIdx]
	delete(box.queues, qid)
	return nil
}

// UpdateQueueStatus TODO
func (box *QueueBox) UpdateQueueStatus(qid QueueID, status QueueStatus) (result Result, err error) {
	box.locker.Lock()
	defer box.locker.Unlock()

	queue, ok := box.queues[qid]

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

	queueIDs := box.queueIDs[oldStatus]
	lastIdx := len(queueIDs) - 1
	lastQueueID := queueIDs[lastIdx]
	lastQueue := box.queues[lastQueueID]
	lastQueue.Idx = queue.Idx
	queueIDs[queue.Idx] = lastQueueID
	queueIDs[lastIdx] = QueueID{}
	box.queueIDs[oldStatus] = queueIDs[:lastIdx]

	queueIDs = box.queueIDs[status]
	queue.Idx = len(queueIDs)
	box.queueIDs[status] = append(queueIDs, qid)
	return
}

// GetQueue TODO
func (box *QueueBox) GetQueue(qid QueueID) (*IdxQueue, bool) {
	box.locker.RLock()
	defer box.locker.RUnlock()
	queue, ok := box.queues[qid]
	return queue, ok
}

// QueueNum TODO
func (box *QueueBox) QueueNum(status QueueStatus) int {
	box.locker.RLock()
	defer box.locker.RUnlock()
	if status == QueueNilStatus {
		return len(box.queues)
	}
	return len(box.queueIDs[status])
}

// Info TODO
func (box *QueueBox) Info() (result Result) {
	box.locker.RLock()
	defer box.locker.RUnlock()
	result = Result{}
	for k, v := range box.queueIDs {
		result[string(k)] = len(v)
	}
	return
}

// OrderedQueues TODO
func (box *QueueBox) OrderedQueues(k int64, start int64, status QueueStatus) Result {
	box.locker.RLock()
	defer box.locker.RUnlock()
	queueIDs := box.queueIDs[status]
	l := len(queueIDs)
	if start < 0 {
		start = 0
	}
	var out []Result
	if l <= 0 || k <= 0 {
		out = []Result{}
	} else {
		for _, qid := range queueIDs[start : start+k] {
			queue, ok := box.queues[qid]
			if ok {
				r := Result{"qid": qid, "qsize": queue.QueueSize()}
				out = append(out, r)
			}
		}
	}
	return Result{
		"k":      k,
		"start":  start,
		"queues": out,
		"total":  l,
		"status": status,
	}
}

// RandomQueues TODO
func (box *QueueBox) RandomQueues(k int64, status QueueStatus) Result {
	box.locker.RLock()
	defer box.locker.RUnlock()
	queueIDs := box.queueIDs[status]
	l := int64(len(queueIDs))
	out := []Result{}
	if l <= 0 || k <= 0 {
	} else {
		if k > l {
			k = l
		}
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		s := r.Perm(int(k)) // TODO not safe
		for _, i := range s {
			qid := queueIDs[i]
			queue, ok := box.queues[qid]
			if ok {
				r := Result{"qid": qid, "qsize": queue.QueueSize()}
				out = append(out, r)
			}
		}
	}
	return Result{
		"k":      k,
		"queues": out,
		"total":  l,
		"status": status,
	}
}
