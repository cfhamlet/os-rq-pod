package serv

import (
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
)

// RedisServ TODO
type RedisServ struct {
	*Serv
	redis *redis.Client
}

// NewRedisServ TODO
func NewRedisServ(conf *viper.Viper, client *redis.Client) *RedisServ {
	return &RedisServ{New(conf), client}
}

// RedisInfo TODO
func (serv *RedisServ) RedisInfo(section ...string) (utils.ParsedRedisInfo, error) {
	info, err := serv.redis.Info(section...).Result()
	parsed := utils.ParseRedisInfo(info)
	return parsed, err
}

// Client TODO
func (serv *RedisServ) Client() *redis.Client {
	return serv.redis
}

// MetaInfo TODO
func (serv *RedisServ) MetaInfo() (result sth.Result, err error) {
	result = serv.Serv.MetaInfo()
	t := time.Now()
	r, err := serv.RedisInfo()
	rinfo := sth.Result{
		"info":      r,
		"_cost_ms_": utils.SinceMS(t),
	}
	result["redis"] = rinfo
	return
}
