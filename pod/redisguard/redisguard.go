package redisguard

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/cfhamlet/os-rq-pod/pod/global"
	"github.com/cfhamlet/os-rq-pod/pod/queuebox"
	"github.com/cfhamlet/os-rq-pod/pod/redisconfig"
	"github.com/go-redis/redis/v7"
	"github.com/spf13/cast"
)

// RedisMemoryLimit TODO
var RedisMemoryLimit = "limit.redis.memory"

// Task TODO
type Task struct {
	guard *Guard
	stop  chan struct{}
}

// NewTask TODO
func NewTask(guard *Guard) *Task {
	return &Task{guard, make(chan struct{})}
}

func (task *Task) run() {
	for {

		info, err := task.guard.client.Info("memory").Result()
		if err != nil {
			log.Logger.Warningf("update redis memory fail", err)
		} else {
			_, memStr := utils.ExtractRedisInfo(info, "used_memory_rss")
			memory, err := cast.ToInt64E(memStr)
			if err != nil {
				log.Logger.Warningf("update redis memory fail", err)
			} else {
				task.guard.updateRedisMemory(memory)
			}
		}
		select {
		case <-task.stop:
		case <-time.After(time.Second):
		}
	}
}

// Start TODO
func (task *Task) Start() {
	go task.run()
}

// Stop TODO
func (task *Task) Stop() {
	close(task.stop)
}

// Guard TODO
type Guard struct {
	*serv.Serv
	queuBox        *queuebox.QueueBox
	client         *redis.Client
	maxRedisMemory int64
	redisMemory    int64
	updateTime     time.Time
	task           *Task
	*sync.RWMutex
}

// New TODO
func New(serv *serv.Serv, queueBox *queuebox.QueueBox, client *redis.Client) *Guard {
	return &Guard{
		serv,
		queueBox,
		client,
		0, 0,
		time.Now(),
		nil,
		&sync.RWMutex{},
	}
}

// OnStart TODO
func (guard *Guard) OnStart(context.Context) error {
	guard.maxRedisMemory = guard.Conf().GetInt64(RedisMemoryLimit)
	err := guard.Serv.Message().Subscribe(redisconfig.ConfigUpdated, guard.updateLimit)
	if err != nil {
		return err
	}
	guard.task = NewTask(guard)
	guard.task.Start()
	return nil
}

// OnStop TODO
func (guard *Guard) OnStop(context.Context) error {
	guard.task.Stop()
	return nil
}

func (guard *Guard) updateRedisMemory(memory int64) {
	guard.Lock()
	defer guard.Unlock()
	guard.redisMemory = memory
	guard.updateTime = time.Now()
	log.Logger.Debugf("update redis memeory %d, max %d",
		memory, guard.maxRedisMemory)
}

func (guard *Guard) updateLimit(key string, value interface{}) {
	guard.Lock()
	defer guard.Unlock()
	if key != RedisMemoryLimit {
		return
	}
	old := guard.maxRedisMemory
	new := cast.ToInt64(value)
	guard.maxRedisMemory = new
	log.Logger.Infof("update %s from %v to %v", RedisMemoryLimit, old, new)
}

// Info TODO
func (guard *Guard) Info() sth.Result {
	guard.RLock()
	defer guard.RUnlock()
	result := sth.Result{
		"memoroy":     guard.redisMemory,
		"max_memoroy": guard.maxRedisMemory,
		"update_time": guard.updateTime.Unix(),
	}
	return result
}

// Guard TODO
func (guard *Guard) Guard() error {
	guard.RLock()
	defer guard.RUnlock()
	if guard.redisMemory > guard.maxRedisMemory {
		return global.ExceedLimitError(
			fmt.Sprintf("redis memory %v, max %v",
				guard.redisMemory, guard.maxRedisMemory))
	}
	return nil
}
