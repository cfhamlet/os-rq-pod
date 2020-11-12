package queuebox

import (
	"context"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/slicemap"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/cfhamlet/os-rq-pod/pod/global"
	"github.com/cfhamlet/os-rq-pod/pod/reqwrap"
	"github.com/cfhamlet/os-rq-pod/pkg/heap"
	"github.com/go-redis/redis/v7"
)

// QueueBox TODO
type QueueBox struct {
	*serv.Serv
	stats        *Stats
	statusQueues map[QueueStatus]*slicemap.Viewer
	client       *redis.Client
	reqWrapper   *reqwrap.RequestWrapper
	*utils.BulkLock
}

// New TODO
func New(serv *serv.Serv, client *redis.Client, reqWrapper *reqwrap.RequestWrapper) *QueueBox {
	statusQueues := map[QueueStatus]*slicemap.Viewer{}
	for _, status := range QueueStatusList {
		statusQueues[status] = slicemap.NewViewer(nil)
	}
	return &QueueBox{
		serv,
		NewStats(),
		statusQueues,
		client,
		reqWrapper,
		utils.NewBulkLock(1024),
	}
}

// CallByQueue TODO
type CallByQueue func(*Queue) (sth.Result, error)

// OnStart TODO
func (box *QueueBox) OnStart(context.Context) error {
	return box.Load()
}

// OnStop TODO
func (box *QueueBox) OnStop(context.Context) error {
	return nil
}

func (box *QueueBox) doOnCoreWorking(f func() (sth.Result, error)) (sth.Result, error) {
	r, e := box.DoWithLockOnWorkStatus(
		func() (interface{}, error) {
			return f()
		}, true, true)
	if r == nil {
		return nil, e
	}
	return r.(sth.Result), e
}

// Load TODO
func (box *QueueBox) Load() (err error) {
	for _, p := range []struct {
		opt  string
		key  string
		pat  string
		qidf func(string) (sth.QueueID, error)
	}{
		{
			"scan",
			"",
			global.RedisQueueKeyPrefix + "*",
			QueueIDFromRedisKey,
		},
		{
			"sscan",
			global.RedisPausedQueuesKey,
			"*",
			QueueIDFromString,
		},
	} {

		scanner := utils.NewScanner(box.client,
			p.opt, p.key, p.pat, 1000)
		log.Logger.Infof("load start %s %s %s", p.opt, p.key, p.pat)
		err = box.load(scanner, p.qidf)
		box.logLoad(err)
		if err != nil {
			return
		}
	}
	return
}

func (box *QueueBox) load(scanner *utils.Scanner, keyToQueueID func(string) (sth.QueueID, error)) (err error) {
	return scanner.Scan(
		func(keys []string) (err error) {
			for _, key := range keys {
				err = box.Serv.SetStatus(serv.Preparing)
				if err != nil {
					break
				}
				qid, e := keyToQueueID(key)
				if e != nil {
					log.Logger.Warning(e)
				} else {
					_, err = box.SyncQueue(qid, true)
					if err != nil {
						break
					}
				}
			}
			return
		},
	)
}

func (box *QueueBox) logLoad(err error) {
	loadQueues := map[QueueStatus]int{}
	for status, queues := range box.statusQueues {
		loadQueues[status] = queues.Size()
	}
	requestNum := box.stats.RequestNum()

	logf := log.Logger.Infof
	args := []interface{}{loadQueues, requestNum, "finish"}
	if err != nil {
		logf = log.Logger.Errorf
		args[2] = err
	}

	logf("load queues: %v requests: %d %s", args...)
}

func (box *QueueBox) withLockMustExist(qid sth.QueueID, f CallByQueue, rLock bool) (result sth.Result, err error) {
	iid := qid.ItemID()
	if rLock {
		box.RLock(iid)
		defer box.RUnlock(iid)
	} else {
		box.Lock(iid)
		defer box.Unlock(iid)
	}
	var item slicemap.Item
	for status := range box.statusQueues {
		item = box.statusQueues[status].Get(iid)
		if item != nil {
			return f(item.(*Queue))
		}
	}
	err = global.NotExistError(utils.Text(qid))
	return
}

func (box *QueueBox) enqueueRequest(req *request.Request, head bool) (result sth.Result, err error) {
	qid := sth.QueueIDFromRequest(req)
	iid := qid.ItemID()
	workingQueues := box.statusQueues[Working]
	pausedQueues := box.statusQueues[Paused]

	box.RLock(iid)
	workingQueues.View(iid, func(item slicemap.Item) {
		if item == nil {
			pausedQueues.View(iid, func(item slicemap.Item) {
				if item != nil {
					err = global.UnavailableError(utils.Text(qid))
				}
			})
		} else {
			queue := item.(*Queue)
			result, err = queue.enqueue(req, head)
		}
	})
	box.RUnlock(iid)

	if result != nil || err != nil {
		return
	}

	box.Lock(iid)
	if workingQueues.Get(iid) != nil ||
		pausedQueues.Get(iid) != nil {
		box.Unlock(iid)
		return box.enqueueRequest(req, head)
	}
	newQueue := NewQueue(box, qid)
	_, err = newQueue.Sync()
	if err != nil {
		box.Unlock(iid)
		return
	}
	if newQueue.Status() == Working {
		result, err = newQueue.enqueue(req, head)
		if err == nil || newQueue.QueueSize() > 0 {
			workingQueues.Add(newQueue)
		}
	} else {
		pausedQueues.Add(newQueue)
		err = global.UnavailableError(utils.Text(qid))
	}
	box.Unlock(iid)
	return
}

func (box *QueueBox) xxEnqueueRequest(req *request.Request, head bool) (result sth.Result, err error) {
	return box.enqueueRequest(req, head)
}

// EnqueueRequest TODO
func (box *QueueBox) EnqueueRequest(req *request.Request, head bool) (result sth.Result, err error) {
	return box.doOnCoreWorking(
		func() (sth.Result, error) {
			return box.xxEnqueueRequest(req, head)
		},
	)
}

func (box *QueueBox) xxDequeueRequest(qid sth.QueueID) (req *request.Request, err error) {
	iid := qid.ItemID()
	deleteIdle := false
	workingQueues := box.statusQueues[Working]
	pausedQueues := box.statusQueues[Paused]

	box.RLock(iid)
	workingQueues.View(iid, func(item slicemap.Item) {
		if item == nil {
			pausedQueues.View(iid, func(item slicemap.Item) {
				reason := utils.Text(qid)
				if item != nil {
					err = global.UnavailableError(reason)
				} else {
					err = global.NotExistError(reason)
				}
			})
		} else {
			var qsize int64
			queue := item.(*Queue)
			req, qsize, err = queue.Dequeue()
			if qsize <= 0 || err == redis.Nil {
				deleteIdle = true
			}
		}
	},
	)
	box.RUnlock(iid)

	if !deleteIdle {
		return
	}
	_, _ = box.SyncQueue(qid, false)
	return
}

// DequeueRequest TODO
func (box *QueueBox) DequeueRequest(qid sth.QueueID) (req *request.Request, err error) {
	r, e := box.DoWithLockOnWorkStatus(
		func() (interface{}, error) {
			return box.xxDequeueRequest(qid)
		}, true, true)
	if r == nil {
		return nil, e
	}
	return r.(*request.Request), e
}

// ClearQueue TODO
func (box *QueueBox) ClearQueue(qid sth.QueueID) (sth.Result, error) {
	return box.withLockMustExist(qid,
		func(queue *Queue) (result sth.Result, err error) {
			result, err = queue.Clear(false)
			if err == nil {
				if queue.QueueSize() <= 0 &&
					queue.Status() == Working {
					box.statusQueues[Working].Delete(queue.ItemID())
				}
			}
			return
		}, false)
}

// DeleteQueue TODO
func (box *QueueBox) DeleteQueue(qid sth.QueueID) (sth.Result, error) {
	return box.withLockMustExist(qid,
		func(queue *Queue) (result sth.Result, err error) {
			result, err = queue.Clear(true)
			box.statusQueues[queue.Status()].Delete(qid.ItemID())
			return
		}, false)
}

// ViewQueue TODO
func (box *QueueBox) ViewQueue(qid sth.QueueID, start int64, end int64) (result sth.Result, err error) {
	return box.withLockMustExist(qid,
		func(queue *Queue) (sth.Result, error) {
			return queue.View(start, end)
		}, true)
}

func (box *QueueBox) fillQueues(iter slicemap.Iterator) []sth.Result {
	out := []sth.Result{}
	iter.Iter(
		func(item slicemap.Item) bool {
			queue := item.(*Queue)
			r := sth.Result{"qid": queue.ID(), "qsize": queue.QueueSize()}
			out = append(out, r)
			return true
		},
	)
	return out
}

// ViewQueues TODO
func (box *QueueBox) ViewQueues(k int, start int, status QueueStatus) sth.Result {
	queues := box.statusQueues[status]
	l := queues.Size()
	var out []sth.Result
	if l <= 0 || k <= 0 {
		out = []sth.Result{}
	} else {
		if start < 0 {
			start = 0
		}
		iterator := slicemap.NewSubIter(queues.Map, start, k)
		out = box.fillQueues(iterator)
	}
	return sth.Result{
		"k":      k,
		"start":  start,
		"queues": out,
		"count":  len(out),
		"total":  l,
		"status": status,
	}
}

func (box *QueueBox) topNQueues(iter slicemap.Iterator, k int) []sth.Result {
	h := heap.NewHeap(func(a interface{}, b interface{}) bool{
			qa:=a.(*Queue)
			qb:=b.(*Queue)
			return qa.QueueSize()<qb.QueueSize()
		},
	)
	iter.Iter(
		func(item slicemap.Item) bool {
			queue := item.(*Queue)
			h.Push(queue)
			if(h.Len()>k){
				h.Pop()
			}
			return true
		},
	)
	out := []sth.Result{}
	for {
		var q interface{}
		if q:=h.Pop();q==nil {
			break
		}
		queue:=q.(*Queue)
		r := sth.Result{"qid": queue.ID(), "qsize": queue.QueueSize()}
		out = append(out, r)
	}
	return out
}

// ViewTopNQueues TODO
func (box *QueueBox) ViewTopNQueues(k int, status QueueStatus) sth.Result {
	queues := box.statusQueues[status]
	l := queues.Size()
	var out []sth.Result
	if l <= 0 || k <= 0 {
		out = []sth.Result{}
	} else {
		iterator := slicemap.NewBaseIter(queues.Map)
		out = box.topNQueues(iterator, k)
	}
	return sth.Result{
		"k":      k,
		"queues": out,
		"count":  len(out),
		"total":  l,
		"status": status,
	}
}

func (box *QueueBox) xxQueues(k int) sth.Result {
	queues := box.statusQueues[Working]
	l := queues.Size()
	var out []sth.Result
	if l <= 0 || k <= 0 {
		out = []sth.Result{}
	} else {
		iterator := slicemap.NewRandomKIter(queues.Map, k)
		out = box.fillQueues(iterator)
	}
	return sth.Result{
		"k":      k,
		"queues": out,
		"count":  len(out),
		"total":  l,
	}
}

// Queues TODO
func (box *QueueBox) Queues(k int) (sth.Result, error) {
	return box.doOnCoreWorking(
		func() (sth.Result, error) {
			return box.xxQueues(k), nil
		},
	)
}

// Info TODO
func (box *QueueBox) Info() (result sth.Result) {
	r := sth.Result{}
	for k, v := range box.statusQueues {
		r[utils.Text(k)] = v.Size()
	}
	result = sth.Result{
		"queues":   r,
		"requests": box.stats.Stats(),
	}
	return
}

// QueueInfo TODO
func (box *QueueBox) QueueInfo(qid sth.QueueID) (sth.Result, error) {
	return box.withLockMustExist(qid,
		func(queue *Queue) (sth.Result, error) {
			return queue.Info(), nil
		}, true)
}

// PauseQueue TODO
func (box *QueueBox) PauseQueue(qid sth.QueueID) (sth.Result, error) {
	return box.SetStatus(qid, Paused)
}

// SetStatus TODO
func (box *QueueBox) SetStatus(qid sth.QueueID, newStatus QueueStatus) (sth.Result, error) {
	return box.withLockMustExist(qid,
		func(queue *Queue) (result sth.Result, err error) {
			oldStatus := queue.Status()
			err = queue.SetStatus(newStatus)
			if err == nil && oldStatus != queue.Status() {
				iid := queue.ItemID()
				box.statusQueues[oldStatus].Delete(iid)
				box.statusQueues[newStatus].Add(queue)
			}
			return queue.Info(), err
		}, false)
}

// ResumeQueue TODO
func (box *QueueBox) ResumeQueue(qid sth.QueueID) (sth.Result, error) {
	return box.SetStatus(qid, Working)
}

func (box *QueueBox) syncQueue(queue *Queue) (result sth.Result, err error) {
	oldStatus := queue.Status()
	result, err = queue.Sync()
	if err != nil {
		return
	}
	iid := queue.ItemID()
	newStatus := queue.Status()
	box.statusQueues[oldStatus].Delete(iid)
	if queue.QueueSize() > 0 || newStatus != Working {
		box.statusQueues[newStatus].Add(queue)
	}

	return
}

// Stats TODO
func (box *QueueBox) Stats() *Stats {
	return box.stats
}

// SyncQueue TODO
func (box *QueueBox) SyncQueue(qid sth.QueueID, force bool) (result sth.Result, err error) {
	iid := qid.ItemID()

	box.Lock(iid)
	defer box.Unlock(iid)

	var item slicemap.Item
	for status := range box.statusQueues {
		item = box.statusQueues[status].Get(iid)
		if item != nil {
			return box.syncQueue(item.(*Queue))
		}
	}

	if !force {
		err = global.NotExistError(qid.String())
		return
	}
	queue := NewQueue(box, qid)
	return box.syncQueue(queue)
}