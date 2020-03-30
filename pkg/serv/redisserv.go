package serv

import (
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
func (serv *RedisServ) RedisInfo(section ...string) (string, error) {
	return serv.redis.Info(section...).Result()
}

// Client TODO
func (serv *RedisServ) Client() *redis.Client {
	return serv.redis
}
