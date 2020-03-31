package pod

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/go-redis/redis/v7"
)

// Queue TODO
type Queue struct {
	box       *QueueBox
	id        sth.QueueID
	redisKey  string
	status    QueueStatus
	qsize     int64
	queuing   int64
	dequeuing int64
}

// Status TODO
func (queue *Queue) Status() QueueStatus {
	return queue.status
}

// SetStatus TODO
func (queue *Queue) SetStatus(newStatus QueueStatus) (err error) {
	oldStatus := queue.status
	if oldStatus == newStatus {
		return
	}
	f := queue.box.core.Client().SAdd
	if oldStatus == Paused {
		f = queue.box.core.Client().SRem
	}
	_, err = f(RedisPausedQueuesKey, queue.ID().String()).Result()
	if err == nil {
		queue.status = newStatus
	}
	return
}

// ID TODO
func (queue *Queue) ID() sth.QueueID {
	return queue.id
}

// ItemID TODO
func (queue *Queue) ItemID() uint64 {
	return queue.id.ItemID()
}

// NewQueue TODO
func NewQueue(box *QueueBox, id sth.QueueID) *Queue {
	return &Queue{
		box,
		id,
		RedisKeyFromQueueID(id),
		Working,
		0, 0, 0,
	}
}

// Sync TODO
func (queue *Queue) Sync() (result sth.Result, err error) {
	result = queue.metaInfo()
	oldSize := queue.qsize
	oldStatus := queue.status

	pipe := queue.box.core.Client().Pipeline()
	pipe.LLen(queue.redisKey)
	pipe.SIsMember(RedisPausedQueuesKey, queue.ID().String())
	var cmders []redis.Cmder
	t := time.Now()
	cmders, err = pipe.Exec()
	result["redis"] = sth.Result{
		"_cost_ms_": utils.SinceMS(t),
	}
	if err != nil {
		return
	}
	cmdQueueLength := cmders[0].(*redis.IntCmd)
	var newSize int64
	newSize, err = cmdQueueLength.Result()
	var paused bool
	if err == nil {
		cmdExist := cmders[1].(*redis.BoolCmd)
		paused, err = cmdExist.Result()
	}
	if err != nil {
		return
	}

	if oldSize != newSize {
		offset := newSize - oldSize
		queue.qsize = newSize
		if offset != 0 {
			queue.box.stats.IncrRequestNum(offset)
		}
	}
	result["osize"] = oldSize
	result["qsize"] = newSize
	if paused {
		queue.status = Paused
	}
	result["ostatus"] = oldStatus
	result["status"] = queue.status
	return
}

func (queue *Queue) updateInput(n int64) int64 {
	queue.box.stats.UpdateInput(n)
	return queue.incr(n)
}

func (queue *Queue) updateOutput(n int64) int64 {
	queue.box.stats.UpdateOutput(n)
	return queue.decr(n)
}

func (queue *Queue) incr(n int64) int64 {
	return atomic.AddInt64(&(queue.qsize), n)
}

func (queue *Queue) decr(n int64) int64 {
	return atomic.AddInt64(&(queue.qsize), 0-n)
}

func (queue *Queue) incrDequeuing(n int64) int64 {
	return atomic.AddInt64(&(queue.dequeuing), n)
}
func (queue *Queue) decrDequeuing(n int64) int64 {
	return atomic.AddInt64(&(queue.dequeuing), 0-n)
}
func (queue *Queue) incrQueuing(n int64) int64 {
	return atomic.AddInt64(&(queue.queuing), n)
}
func (queue *Queue) decrQueuing(n int64) int64 {
	return atomic.AddInt64(&(queue.queuing), 0-n)
}

// QueueSize TODO
func (queue *Queue) QueueSize() int64 {
	return atomic.LoadInt64(&queue.qsize)
}

// Dequeuing TODO
func (queue *Queue) Dequeuing() int64 {
	return atomic.LoadInt64(&queue.dequeuing)
}

// Queuing TODO
func (queue *Queue) Queuing() int64 {
	return atomic.LoadInt64(&queue.queuing)
}

// Push TODO
func (queue *Queue) Push(request *request.Request) (result sth.Result, err error) {

	queuing := queue.incrQueuing(1)
	defer queue.decrQueuing(1)

	if queuing > 198405 {
		err = UnavailableError(fmt.Sprintf("%s too many push requests %d", queue.ID(), queuing))
		return
	}

	if request.Meta == nil {
		request.Meta = make(map[string]interface{})
	}
	request.Meta["_pod_in_"] = time.Now().Unix()
	j, err := json.Marshal(request)
	if err == nil {
		rsize, err := queue.box.core.Client().RPush(queue.redisKey, j).Result()
		if err == nil {
			qsize := queue.updateInput(1)
			result = sth.Result{"rsize": rsize, "qsize": qsize}
		}
	}
	return
}

// Pop TODO
func (queue *Queue) Pop() (req *request.Request, qsize int64, err error) {

	dequeuing := queue.incrDequeuing(1)
	defer queue.decrDequeuing(1)
	qsize = queue.QueueSize()

	if dequeuing > qsize || dequeuing > 198405 {
		msg := fmt.Sprintf("%s qsize %d, dequeuing %d", queue.ID(), qsize, dequeuing)
		log.Logger.Debug(msg)
		err = UnavailableError(msg)
		return
	}

	var r string
	r, err = queue.box.core.Client().LPop(queue.redisKey).Result()
	if err == nil {
		qsize = queue.updateOutput(1)
		req = &request.Request{}
		err = json.Unmarshal([]byte(r), req)
	}
	return
}

// metaInfo TODO
func (queue *Queue) metaInfo() sth.Result {
	return sth.Result{
		"rkey":      queue.redisKey,
		"qid":       queue.ID(),
		"status":    queue.status,
		"qsize":     queue.QueueSize(),
		"queuing":   queue.Queuing(),
		"dequeuing": queue.Dequeuing(),
	}
}

// View TODO
func (queue *Queue) View(start int64, end int64) (result sth.Result, err error) {

	t := time.Now()
	var requests []string
	requests, err = queue.box.core.Client().LRange(queue.redisKey, start, end).Result()
	result = sth.Result{}
	result["redis"] = sth.Result{
		"_cost_ms_": utils.SinceMS(t),
	}
	result["requests"] = requests
	result["lrange"] = []int64{start, end}
	return
}

// Info TODO
func (queue *Queue) Info() sth.Result {
	return queue.metaInfo()
}

// Clear TODO
func (queue *Queue) Clear(erase bool) (result sth.Result, err error) {

	result = queue.metaInfo()
	result["drop"] = 0

	t := time.Now()
	err = queue.box.core.Client().Watch(func(tx *redis.Tx) error {
		drop, e := tx.LLen(queue.redisKey).Result()
		if e == nil {
			_, e = tx.Del(queue.redisKey).Result()
		}
		if e == nil && drop != 0 {
			result["drop"] = drop
		}
		if e == nil && erase {
			_, e = tx.SRem(RedisPausedQueuesKey, queue.ID().String()).Result()
		}
		return e
	}, queue.redisKey)
	result["redis"] = sth.Result{
		"_cost_ms_": utils.SinceMS(t),
	}

	if err == nil && queue.qsize != 0 {
		queue.box.stats.IncrRequestNum(0 - queue.qsize)
		queue.qsize = 0
	}

	return
}
