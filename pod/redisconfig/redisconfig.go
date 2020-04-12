package redisconfig

import (
	"context"
	"fmt"
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pod/global"
	"github.com/go-redis/redis/v7"
	"github.com/spf13/cast"
)

// ConfigUpdated TODO
var ConfigUpdated = "ConfigUpdated"

// RedisConfig TODO
type RedisConfig struct {
	*serv.Serv
	client *redis.Client
	online map[string]bool
	*sync.RWMutex
}

// New TODO
func New(serv *serv.Serv, client *redis.Client, online []string) *RedisConfig {
	ks := map[string]bool{}
	for _, key := range online {
		ks[key] = true
	}
	return &RedisConfig{serv, client, ks, &sync.RWMutex{}}
}

// OnStart TODO
func (rds *RedisConfig) OnStart(ctx context.Context) (err error) {
	var j string
	j, err = rds.client.Get(global.RedisConfigKey).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil
			log.Logger.Warning("no online config")
		}
		return
	}
	configs := map[string]interface{}{}
	err = json.Unmarshal([]byte(j), &configs)
	if err != nil {
		return
	}
	rds.update(configs, false)
	return nil
}

// OnStop TODO
func (rds *RedisConfig) OnStop(ctx context.Context) error {
	return nil
}

// GetConfig TODO
func (rds *RedisConfig) GetConfig(key string) (interface{}, error) {
	rds.RLock()
	defer rds.RUnlock()
	exist := rds.Serv.Conf().IsSet(key)
	if !exist {
		return nil, global.NotExistError("key")
	}
	return rds.Serv.Conf().Get(key), nil
}

// AllConfigs TODO
func (rds *RedisConfig) AllConfigs() sth.Result {
	rds.RLock()
	defer rds.RUnlock()
	result := sth.Result{}
	for _, key := range rds.Conf().AllKeys() {
		result[key] = rds.Conf().Get(key)
	}
	return result
}

// SetConfigs TODO
func (rds *RedisConfig) SetConfigs(kv map[string]interface{}) (result sth.Result, err error) {
	rds.Lock()
	defer rds.Unlock()
	result = sth.Result{}
	castkv := sth.Result{}
	for k, v := range kv {
		r, e := rds.checkAndCast(k, v)
		t := sth.Result{}
		if e != nil {
			t["err"] = e
			err = global.UnavailableError("something wrong")
		} else {
			old := rds.Conf().Get(k)
			if r != old {
				castkv[k] = r
			}
			t["value"] = r
			t["old"] = rds.Conf().Get(k)
		}
		result[k] = t
	}

	if err == nil {
		err = rds.save(castkv)
		if err == nil {
			rds.update(castkv, true)
		}
	}

	return
}

func (rds *RedisConfig) save(kv map[string]interface{}) (err error) {
	if len(kv) <= 0 {
		return
	}
	var keys []string
	if len(rds.online) <= 0 {
		keys = rds.Conf().AllKeys()
	} else {
		for key := range rds.online {
			keys = append(keys, key)
		}
	}
	result := map[string]interface{}{}
	for _, key := range keys {
		value, ok := kv[key]
		if ok {
			result[key] = value
		} else {
			result[key] = rds.Conf().Get(key)
		}
	}
	var j []byte
	j, err = json.Marshal(result)
	if err == nil {
		_, err = rds.client.Set(global.RedisConfigKey, j, 0).Result()
	}
	return
}

func (rds *RedisConfig) update(kv map[string]interface{}, sendMsg bool) {
	for k, v := range kv {
		old := rds.Conf().Get(k)
		if old != v {
			rds.Conf().Set(k, v)
			if sendMsg {
				log.Logger.Debug("send msg", ConfigUpdated, k, v)
				rds.Serv.Message().Publish(ConfigUpdated, k, v)
			}
		}
	}
}

// SetConfig TODO
func (rds *RedisConfig) checkAndCast(key string, value interface{}) (castValue interface{}, err error) {
	if len(rds.online) > 0 {
		_, ok := rds.online[key]
		if !ok {
			err = global.NotExistError("not online config key: " + key)
			return
		}
	}
	conf := rds.Conf()
	if !conf.IsSet(key) {
		err = global.NotExistError("not exist: " + key)
		return
	}

	oldValue := conf.Get(key)
	castValue = nil
	switch oldValue.(type) {
	case int:
		castValue, err = cast.ToIntE(value)
	case int64:
		castValue, err = cast.ToInt64E(value)
	case string:
		castValue, err = cast.ToStringE(value)
	case bool:
		castValue, err = cast.ToBoolE(value)
	case float64:
		castValue, err = cast.ToFloat64E(value)
	default:
		err = fmt.Errorf(fmt.Sprintf("not support type: %T", oldValue))
	}

	return
}
