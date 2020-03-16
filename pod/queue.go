package pod

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/go-redis/redis/v7"
	"github.com/segmentio/fasthash/fnv1a"
)

// QueueStatus type
type QueueStatus string

// QueueStatus enum
const (
	QueueUndefined QueueStatus = "undefined"
	QueueWorking   QueueStatus = "working"
	QueuePaused    QueueStatus = "paused"
)

// QueueStatusMap TODO
var QueueStatusMap = map[string]QueueStatus{
	string(QueueUndefined): QueueUndefined,
	string(QueueWorking):   QueueWorking,
	string(QueuePaused):    QueuePaused,
}

// QueueStatusList TODO
var QueueStatusList = []QueueStatus{
	QueueUndefined,
	QueueWorking,
	QueuePaused,
}

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

// QueueIDFromRedisKey TODO
func QueueIDFromRedisKey(key string) (qid QueueID, err error) {
	if !strings.HasPrefix(key, RedisQueueKeyPrefix) {
		err = fmt.Errorf(`invalid redis key %s, not starts with "%s"`, key, RedisQueueKeyPrefix)
	} else {
		parts := strings.Split(key[len(RedisQueueKeyPrefix):], ":")
		if len(parts) != 3 {
			err = fmt.Errorf(`invalid redis key %s, not "%shost:port:scheme"`, key, RedisQueueKeyPrefix)
		} else {
			qid = QueueID{parts[0], parts[1], parts[2]}
		}
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
	return &Queue{pod, id, status, RedisKeyFromQueueID(id), 0, &sync.RWMutex{}, time.Now(), 0, 0}
}

// Sync TODO
func (queue *Queue) Sync(lock bool) (result Result, err error) {
	if lock {
		queue.locker.Lock()
		defer queue.locker.Unlock()
	}

	result = queue.metaInfo()
	oldSize := queue.qsize
	newSize, err := queue.pod.Client.LLen(queue.redisKey).Result()

	if err != nil {
		return
	}

	result["osize"] = oldSize
	if oldSize != newSize {
		offset := newSize - oldSize
		queue.qsize = newSize
		result["qsize"] = newSize
		if offset != 0 {
			queue.pod.stats.IncrRequestNum(offset)
		}
	}
	return
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
func (queue *Queue) Put(request *request.Request) (Result, error) {
	queue.locker.RLock()
	defer queue.locker.RUnlock()

	if queue.status != QueueWorking {
		return nil, UnavailableError(fmt.Sprintf("%s %s", queue.ID, queue.status))
	}

	queuing := queue.incrQueuing(1)
	defer queue.decrQueuing(1)

	if queuing > 198405 {
		return nil, UnavailableError(fmt.Sprintf("%s too many put requests %d", queue.ID, queuing))
	}

	if request.RawReq.Meta == nil {
		request.RawReq.Meta = make(map[string]interface{})
	}
	request.RawReq.Meta["_pod_in_"] = time.Now().Unix()
	j, err := request.JSON()
	if err != nil {
		return nil, err
	}
	rsize, err := queue.pod.Client.RPush(queue.redisKey, j).Result()
	if err != nil {
		return nil, err
	}
	qsize := queue.updateInput(1)

	return Result{"rsize": rsize, "qsize": qsize}, err
}

// Get TODO
func (queue *Queue) Get() (result Result, err error) {
	queue.locker.RLock()
	defer queue.locker.RUnlock()

	if queue.status != QueueWorking {
		err = UnavailableError(fmt.Sprintf("%s %s", queue.ID, queue.status))
		return
	}
	dequeuing := queue.incrDequeuing(1)
	defer queue.decrDequeuing(1)
	qsize := queue.QueueSize()

	if dequeuing > qsize || dequeuing > 198405 {
		err = UnavailableError(fmt.Sprintf("%s qsize %d, dequeuing %d", queue.ID, qsize, dequeuing))
		return
	}

	r, err := queue.pod.Client.LPop(queue.redisKey).Result()
	if err != nil {
		return
	}
	queue.updateOutput(1)
	result = Result{}
	err = json.Unmarshal([]byte(r), &result)
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

	result, err = queue.Sync(false)
	if err == nil {
		var requests []string
		requests, err = queue.pod.Client.LRange(queue.redisKey, start, end).Result()
		result["requests"] = requests
	}
	return
}

// Info TODO
func (queue *Queue) Info() (result Result, err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	result = queue.metaInfo()
	return
}

// Status TODO
func (queue *Queue) Status() QueueStatus {
	queue.locker.RLock()
	defer queue.locker.RUnlock()
	return queue.status
}

// SetStatus TODO
func (queue *Queue) SetStatus(status QueueStatus) (result Result, err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()

	if status == queue.status {
		result = queue.metaInfo()
	} else {
		oldStatus := queue.status
		queue.status = status
		result, err = queue.Sync(false)
		if err != nil {
			queue.status = oldStatus
			result["status"] = oldStatus
		}
	}

	return
}

// Idle TODO
func (queue *Queue) Idle() bool {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	return queue.qsize <= 0 && queue.status == QueueWorking && queue.queuing <= 0
}

// Clear TODO
func (queue *Queue) Clear() (result Result, err error) {
	queue.locker.Lock()
	defer queue.locker.Unlock()

	result = queue.metaInfo()
	result["drop"] = 0

	err = queue.pod.Client.Watch(func(tx *redis.Tx) error {
		drop, err := tx.LLen(queue.redisKey).Result()
		if err == nil {
			_, err = tx.Del(queue.redisKey).Result()
		}
		if err == nil && drop != 0 {
			result["drop"] = drop
		}
		return err
	}, queue.redisKey)

	if err == nil && queue.qsize != 0 {
		queue.pod.stats.IncrRequestNum(0 - queue.qsize)
		queue.qsize = 0
	}

	return
}
