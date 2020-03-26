package pod

import (
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/slicemap"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
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

// LoadQueues TODO
func (box *QueueBox) LoadQueues() (err error) {
	box.Lock()
	defer box.Unlock()

	log.Logger.Info("load queues start")

	scanner := utils.NewScanner(box.pod.Client, "scan", "", RedisQueueKeyPrefix+"*", 2000)
	err = scanner.Scan(
		func(keys []string) (err error) {
			for _, key := range keys {
				err = box.pod.setStatus(Preparing)
				if err != nil {
					break
				}
				qid, e := QueueIDFromRedisKey(key)
				if e != nil {
					log.Logger.Warning("invalid key", key, e)
				} else {
					_, err = box.addQueue(qid)
				}
				if err != nil {
					break
				}
			}
			return
		},
	)

	loadQueues := len(box.queues)
	requestNum := box.pod.stats.RequestNum()

	if err == nil {
		log.Logger.Infof("load queues finish, queues %d, requests %d",
			loadQueues, requestNum)
	} else {
		log.Logger.Errorf("load queues fail, queues %d, requests %d, %s",
			loadQueues, requestNum, err)
	}

	return
}

// LoadPaused TODO
func (box *QueueBox) LoadPaused() (err error) {
	box.Lock()
	defer box.Unlock()

	scanner := utils.NewScanner(box.pod.Client, "sscan", RedisPausedQueuesKey, "*", 1000)
	log.Logger.Info("load paused start")

	err = scanner.Scan(
		func(keys []string) (err error) {
			for _, key := range keys {
				err = box.pod.setStatus(Preparing)
				if err != nil {
					break
				}
				qid, e := QueueIDFromString(key)
				if e != nil {
					log.Logger.Warning(e)
					continue
				}
				var queue *Queue
				queue, err = box.addQueue(qid)
				if err != nil {
					break
				}
				if queue.Status() != QueuePaused {
					err = queue.SetStatus(QueuePaused)
				}
				if err != nil {
					break
				}
			}
			return
		},
	)

	paused := box.statusQueueIDs[QueuePaused].Size()

	if err == nil {
		log.Logger.Infof("load paused finish, paused %d", paused)
	} else {
		log.Logger.Errorf("load paused fail, paused %d, %s", paused, err)
	}

	return
}

// CallByQueue TODO
type CallByQueue func(*Queue) (Result, error)

func (box *QueueBox) withRLockMustExist(qid QueueID, f CallByQueue) (Result, error) {
	return box.withLockRLockMustExist(qid, f, false)
}

func (box *QueueBox) withLockMustExist(qid QueueID, f CallByQueue) (Result, error) {
	return box.withLockRLockMustExist(qid, f, true)
}

func (box *QueueBox) withLockRLockMustExist(qid QueueID, f CallByQueue, lock bool) (Result, error) {
	if lock {
		box.Lock()
		defer box.Unlock()
	} else {
		box.RLock()
		defer box.RUnlock()
	}
	return box.mustExist(qid, f)
}

func (box *QueueBox) mustExist(qid QueueID, f CallByQueue) (result Result, err error) {
	queue, ok := box.queues[qid]
	if !ok {
		err = NotExistError(qid.String())
		return
	}
	return f(queue)
}

// DeleteIdleQueue TODO
func (box *QueueBox) DeleteIdleQueue(qid QueueID) error {
	_, e := box.withLockMustExist(qid,
		func(queue *Queue) (result Result, err error) {
			if queue.Idle() {
				err = queue.SetStatus(QueueRemoved)
			}
			return
		},
	)
	return e
}

// SyncQueue TODO
func (box *QueueBox) SyncQueue(qid QueueID, force bool) (result Result, err error) {
	box.RLock()

	queue, ok := box.queues[qid]
	if !ok {
		if force {
			queue, err = box.addQueue(qid)
		} else {
			err = NotExistError(qid.String())
			box.RUnlock()
			return
		}
	}

	if err == nil {
		if ok {
			result, err = queue.Sync()
		}
		idle := queue.Idle()
		box.RUnlock()
		if idle {
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
		queue, err = box.addQueue(qid)
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
func (box *QueueBox) GetRequest(qid QueueID) (req *request.Request, err error) {
	box.RLock()
	queue, ok := box.queues[qid]

	if !ok {
		err = NotExistError(qid.String())
		box.RUnlock()
		return
	}

	var qsize int64
	req, qsize, err = queue.Get()
	box.RUnlock()
	if qsize <= 0 || err == redis.Nil {
		_, _ = box.SyncQueue(qid, false)
		if err == redis.Nil {
			err = NotExistError(qid.String())
		}
	}
	return
}

func (box *QueueBox) addQueue(qid QueueID) (queue *Queue, err error) {
	box.cLocker.Lock()
	defer box.cLocker.Unlock()

	var ok bool
	queue, ok = box.queues[qid]
	if ok {
		return
	}
	queue = NewQueue(box.pod, qid, QueueInit)
	_, err = queue.Sync()
	newStatus := QueueWorking
	if err != nil {
		newStatus = QueueRemoved
	}
	err = queue.SetStatusOn(newStatus, QueueInit)

	return
}

// DeleteQueue TODO
func (box *QueueBox) DeleteQueue(qid QueueID) (Result, error) {
	return box.withLockMustExist(qid,
		func(queue *Queue) (result Result, err error) {
			result, err = queue.Clear()
			if err == nil {
				err = queue.SetStatus(QueueRemoved)
				if err == nil {
					result["status"] = queue.Status()
				}
			}
			return
		},
	)
}

// ViewQueue TODO
func (box *QueueBox) ViewQueue(qid QueueID, start int64, end int64) (result Result, err error) {
	return box.withRLockMustExist(qid,
		func(queue *Queue) (Result, error) {
			return queue.View(start, end)
		},
	)
}

// ClearQueue TODO
func (box *QueueBox) ClearQueue(qid QueueID) (Result, error) {
	return box.withRLockMustExist(qid,
		func(queue *Queue) (Result, error) {
			return queue.Clear()
		},
	)
}

// SetStatus TODO
func (box *QueueBox) SetStatus(qid QueueID, newStatus QueueStatus) (Result, error) {
	return box.withLockMustExist(qid,
		func(queue *Queue) (result Result, err error) {
			err = queue.SetStatus(newStatus)
			if err == nil {
				result = queue.Info()
			}
			return
		},
	)
}

// QueueInfo TODO
func (box *QueueBox) QueueInfo(qid QueueID) (Result, error) {
	return box.withRLockMustExist(qid,
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
		"count":  len(out),
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
		"count":  len(out),
		"total":  l,
	}
}

// QueuesCount TODO
func (box *QueueBox) QueuesCount(status QueueStatus) int {
	box.Lock()
	defer box.Unlock()
	if status == QueueInit {
		return len(box.queues)
	}
	return box.statusQueueIDs[status].Size()
}

func (box *QueueBox) info() (result Result) {
	r := Result{}
	for k, v := range box.statusQueueIDs {
		r[utils.Text(k)] = v.Size()
	}
	result = Result{
		"total":      len(box.queues),
		"status_num": r,
	}
	return
}

// Info TODO
func (box *QueueBox) Info() (result Result) {
	box.RLock()
	defer box.RUnlock()
	return box.info()
}
