package pod

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/go-redis/redis/v7"
	"github.com/segmentio/fasthash/fnv1a"
)

// QueueID TODO
type QueueID struct {
	Host   string `json:"host"`
	Port   string `json:"port"`
	Scheme string `json:"scheme"`
}

func (qid QueueID) String() string {
	return strings.Join([]string{qid.Host, qid.Port, qid.Scheme}, ":")
}

// ItemID TODO
func (qid QueueID) ItemID() uint64 {
	return fnv1a.HashString64(qid.String())
}

// MarshalJSON TODO
func (qid QueueID) MarshalJSON() ([]byte, error) {
	return json.Marshal(qid.String())
}

// Queue TODO
type Queue struct {
	pod        *Pod
	ID         QueueID
	status     QueueStatus
	redisKey   string
	qsize      int64
	locker     *sync.RWMutex
	createTime time.Time
	queuing    int64
	dequeuing  int64
}

// RedisKeyFromQueueID TODO
func RedisKeyFromQueueID(qid QueueID) string {
	return RedisQueueKeyPrefix + qid.String()
}

// QueueIDFromString TODO
func QueueIDFromString(key string) (qid QueueID, err error) {
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		err = fmt.Errorf(`invalid qid %s, not "host:port:scheme"`, key)
	} else {
		qid = QueueID{parts[0], parts[1], parts[2]}
	}
	return
}

// QueueIDFromRedisKey TODO
func QueueIDFromRedisKey(key string) (qid QueueID, err error) {
	if !strings.HasPrefix(key, RedisQueueKeyPrefix) {
		err = fmt.Errorf(`invalid redis key %s, not starts with "%s"`, key, RedisQueueKeyPrefix)
	} else {
		qid, err = QueueIDFromString(key[len(RedisQueueKeyPrefix):])
	}

	return
}

// QueueIDFromRequest TODO
func QueueIDFromRequest(req *request.Request) QueueID {
	return CreateQueueID(req.Host, req.Port, req.Parsed.Scheme)
}

// CreateQueueID TODO
func CreateQueueID(host, port, scheme string) QueueID {
	defaultPort, ok := utils.DefaultSchemePort[scheme]
	if ok {
		if defaultPort == port {
			port = ""
		}
	}
	return QueueID{Host: host, Port: port, Scheme: scheme}
}

// NewQueue TODO
func NewQueue(pod *Pod, id QueueID, status QueueStatus) *Queue {
	return &Queue{
		pod,
		id,
		status,
		RedisKeyFromQueueID(id),
		0,
		&sync.RWMutex{},
		time.Now(),
		0,
		0,
	}
}

func (queue *Queue) sync() (result Result, err error) {
	result = queue.metaInfo()
	oldSize := queue.qsize
	oldStatus := queue.status

	pipe := queue.pod.Client.Pipeline()
	pipe.LLen(queue.redisKey)
	pipe.SIsMember(RedisPausedQueuesKey, queue.ID.String())
	var cmders []redis.Cmder
	t := time.Now()
	cmders, err = pipe.Exec()
	result["redis"] = Result{
		"_cost_ms": utils.SinceMS(t),
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
			queue.pod.stats.IncrRequestNum(offset)
		}
	}
	result["osize"] = oldSize
	result["qsize"] = newSize
	if paused {
		err = queue.setStatus(QueuePaused)
	}
	if err == nil {
		result["ostatus"] = oldStatus
		result["status"] = queue.status
	}
	return
}

// Sync TODO
func (queue *Queue) Sync() (result Result, err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	return queue.sync()
}

func (queue *Queue) updateInput(n int64) int64 {
	queue.pod.stats.UpdateInput(n)
	return queue.incr(n)
}

func (queue *Queue) updateOutput(n int64) int64 {
	queue.pod.stats.UpdateOutput(n)
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

// Put TODO
func (queue *Queue) Put(request *request.Request) (result Result, err error) {
	queue.locker.RLock()
	defer queue.locker.RUnlock()

	if queue.status != QueueWorking {
		err = UnavailableError(fmt.Sprintf("%s %s", queue.ID, queue.status))
		return
	}

	queuing := queue.incrQueuing(1)
	defer queue.decrQueuing(1)

	if queuing > 198405 {
		err = UnavailableError(fmt.Sprintf("%s too many put requests %d", queue.ID, queuing))
		return
	}

	if request.Meta == nil {
		request.Meta = make(map[string]interface{})
	}
	request.Meta["_pod_in"] = time.Now().Unix()
	j, err := json.Marshal(request)
	if err == nil {
		rsize, err := queue.pod.Client.RPush(queue.redisKey, j).Result()
		if err == nil {
			qsize := queue.updateInput(1)
			result = Result{"rsize": rsize, "qsize": qsize}
		}
	}
	return
}

// Get TODO
func (queue *Queue) Get() (req *request.Request, qsize int64, err error) {
	queue.locker.RLock()
	defer queue.locker.RUnlock()

	if queue.status != QueueWorking {
		err = UnavailableError(fmt.Sprintf("%s %s", queue.ID, queue.status))
		return
	}
	dequeuing := queue.incrDequeuing(1)
	defer queue.decrDequeuing(1)
	qsize = queue.QueueSize()

	if dequeuing > qsize || dequeuing > 198405 {
		msg := fmt.Sprintf("%s qsize %d, dequeuing %d", queue.ID, qsize, dequeuing)
		log.Logger.Debug(msg)
		err = UnavailableError(msg)
		return
	}

	var r string
	r, err = queue.pod.Client.LPop(queue.redisKey).Result()
	if err == nil {
		qsize = queue.updateOutput(1)
		req = &request.Request{}
		err = json.Unmarshal([]byte(r), req)
	}
	return
}

// metaInfo TODO
func (queue *Queue) metaInfo() Result {
	return Result{
		"rkey":      queue.redisKey,
		"qid":       queue.ID,
		"status":    queue.status,
		"qsize":     queue.qsize,
		"ctime":     queue.createTime,
		"queuing":   queue.queuing,
		"dequeuing": queue.dequeuing,
	}
}

// View TODO
func (queue *Queue) View(start int64, end int64) (result Result, err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()

	result, err = queue.sync()
	if err != nil {
		return
	}
	t := time.Now()
	var requests []string
	requests, err = queue.pod.Client.LRange(queue.redisKey, start, end).Result()
	result["redis"] = Result{
		"_cost_ms": utils.SinceMS(t),
	}
	result["requests"] = requests
	result["lrange"] = []int64{start, end}
	return
}

// Info TODO
func (queue *Queue) Info() Result {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	return queue.metaInfo()
}

// Idle TODO
func (queue *Queue) Idle() bool {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	return queue.qsize <= 0 &&
		(queue.status == QueueWorking || queue.status == QueueInit) &&
		queue.queuing <= 0
}

// Clear TODO
func (queue *Queue) Clear() (result Result, err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()

	result = queue.metaInfo()
	result["drop"] = 0

	t := time.Now()
	err = queue.pod.Client.Watch(func(tx *redis.Tx) error {
		drop, e := tx.LLen(queue.redisKey).Result()
		if e == nil {
			_, e = tx.Del(queue.redisKey).Result()
		}
		if e == nil && drop != 0 {
			result["drop"] = drop
		}
		return e
	}, queue.redisKey)
	result["redis"] = Result{
		"_cost_ms": utils.SinceMS(t),
	}

	if err == nil && queue.qsize != 0 {
		queue.pod.stats.IncrRequestNum(0 - queue.qsize)
		queue.qsize = 0
	}

	return
}

// Status TODO
func (queue *Queue) Status() QueueStatus {
	queue.locker.RLock()
	defer queue.locker.RUnlock()
	return queue.status
}

func (queue *Queue) setStatus(newStatus QueueStatus) (err error) {
	oldStatus := queue.status
	if oldStatus == newStatus {
		return
	}
	e := UnavailableError(oldStatus)
	switch oldStatus {
	case QueueInit:
	case QueuePaused:
		switch newStatus {
		case QueueInit:
			err = e
		}
	case QueueWorking:
		switch newStatus {
		case QueueInit:
			err = e
		}
	case QueueRemoved:
		err = e
	}
	if err != nil {
		return
	}
	if newStatus == QueuePaused {
		_, err = queue.pod.Client.SAdd(RedisPausedQueuesKey, queue.ID.String()).Result()
	} else if oldStatus == QueuePaused {
		_, err = queue.pod.Client.SRem(RedisPausedQueuesKey, queue.ID.String()).Result()
	}

	if err != nil {
		return
	}

	box := queue.pod.queueBox
	queue.status = newStatus
	box.statusQueueIDs[oldStatus].Delete(queue.ID)
	if newStatus != QueueRemoved {
		box.queues[queue.ID] = queue
		box.statusQueueIDs[newStatus].Add(queue.ID)
	} else {
		delete(box.queues, queue.ID)
	}
	return
}

// SetStatus TODO
func (queue *Queue) SetStatus(status QueueStatus) (err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	return queue.setStatus(status)
}

// SetStatusOn TODO
func (queue *Queue) SetStatusOn(newStatus QueueStatus, on QueueStatus) (err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	if queue.status == on {
		err = queue.setStatus(newStatus)
	}
	return
}
