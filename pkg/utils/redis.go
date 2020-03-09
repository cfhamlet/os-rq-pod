package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
)

// NewRedisClient create redis client and connect
func NewRedisClient(conf *viper.Viper) (*redis.Client, error) {

	var err error
	client := redis.NewClient(
		&redis.Options{
			Addr:     conf.GetString("redis.addr"),
			Password: conf.GetString("redis.password"),
			DB:       conf.GetInt("redis.db"),
		})

	_, err = client.Ping().Result()
	if err == nil {
		log.Logger.Infof("connect redis succ %s\n", client.Options().Addr)
		debug := false
		if conf.IsSet("debug") {
			debug = conf.GetBool("debug")
		}
		if debug {
			if info, err := client.Info().Result(); err == nil {
				fmt.Fprintf(os.Stderr, "Redis Info:\n%s\n", info)
			}
		}
	} else {
		log.Logger.Errorf("connect redis fail %s %s", client.Options().Addr, err)
	}
	return client, err
}

// ParseRedisInfo TODO
func ParseRedisInfo(info string, key string) (string, string) {
	t := strings.Index(info, key)
	if t < 0 {
		return "", ""
	}
	info = info[t:]
	t = strings.Index(info, ":")
	if t < 0 {
		return "", ""
	}
	k := info[:t]
	info = info[t+1:]

	t = strings.Index(info, "\r\n")
	if t < 0 {
		return "", ""
	}
	v := info[:t]
	return k, v
}
