package pod

import (
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/slicemap"
)

// QueueBox TODO
type QueueBox struct {
	pod            *Pod
	queueMap       map[QueueID]*Queue
	statusQueueIDs map[QueueStatus]*slicemap.Map
	locker         *sync.RWMutex
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
	}
}

// AddQueue TODO
func (box *QueueBox) AddQueue(qid QueueID) (queue *Queue, err error) {
	box.locker.Lock()
	defer box.locker.Unlock()

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

// RemoveQueue TODO
func (box *QueueBox) RemoveQueue(qid QueueID) error {
	box.locker.Lock()
	defer box.locker.Unlock()

	queue, ok := box.queueMap[qid]
	if !ok {
		return QueueNotExist
	}

	status := queue.Status()
	delete(box.queueMap, qid)
	box.statusQueueIDs[status].Delete(qid)
	return nil
}

// UpdateQueueStatus TODO
func (box *QueueBox) UpdateQueueStatus(qid QueueID, status QueueStatus) (result Result, err error) {
	box.locker.Lock()
	defer box.locker.Unlock()

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

// GetQueue TODO
func (box *QueueBox) GetQueue(qid QueueID) (*Queue, bool) {
	box.locker.RLock()
	defer box.locker.RUnlock()
	queue, ok := box.queueMap[qid]
	return queue, ok
}

// QueueNum TODO
func (box *QueueBox) QueueNum(status QueueStatus) int {
	box.locker.RLock()
	defer box.locker.RUnlock()
	if status == QueueUndefined {
		return len(box.queueMap)
	}
	return box.statusQueueIDs[status].Size()
}

// Info TODO
func (box *QueueBox) Info() (result Result) {
	box.locker.RLock()
	defer box.locker.RUnlock()
	result = Result{}
	for k, v := range box.statusQueueIDs {
		result[string(k)] = v.Size()
	}
	return
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
	box.locker.RLock()
	defer box.locker.RUnlock()
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
	box.locker.RLock()
	defer box.locker.RUnlock()
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
