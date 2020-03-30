package serv

import (
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
func (serv *RedisServ) RedisInfo(section ...string) (interface{}, error) {
	info, err := serv.redis.Info(section...).Result()
	parsed := utils.ParseRedisInfo(info)
	return parsed, err
}

// Client TODO
func (serv *RedisServ) Client() *redis.Client {
	return serv.redis
}
