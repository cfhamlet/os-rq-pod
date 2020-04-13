package redisguard

import (
	"context"
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pod/queuebox"
	"github.com/cfhamlet/os-rq-pod/pod/redisconfig"
	"github.com/go-redis/redis/v7"
	"github.com/spf13/cast"
)

// RedisMemoryLimit TODO
var RedisMemoryLimit = "limit.redis.memory"

// Guard TODO
type Guard struct {
	*serv.Serv
	queuBox        *queuebox.QueueBox
	client         *redis.Client
	maxRedisMemory int64
	redisMemory    int64
	*sync.RWMutex
}

// New TODO
func New(serv *serv.Serv, queueBox *queuebox.QueueBox, client *redis.Client) *Guard {
	return &Guard{serv, queueBox, client, 0, 0, &sync.RWMutex{}}
}

// OnStart TODO
func (guard *Guard) OnStart(context.Context) error {
	guard.maxRedisMemory = guard.Conf().GetInt64(RedisMemoryLimit)
	guard.Serv.Message().Subscribe(redisconfig.ConfigUpdated, guard.updateLimit)
	return nil
}

// OnStop TODO
func (guard *Guard) OnStop(context.Context) error {
	return nil
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
	result := sth.Result{}
	return result
}

// Guard TODO
func (guard *Guard) Guard() error {
	guard.RLock()
	defer guard.RUnlock()
	return nil
}
