package controllers

import (
	"strings"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v7"
)

// RedisController TODO
type RedisController struct {
	client *redis.Client
}

// NewRedisController TODO
func NewRedisController(client *redis.Client) *RedisController {
	return &RedisController{client}
}

// RedisInfo TODO
func (ctrl *RedisController) RedisInfo(c *gin.Context) (sth.Result, error) {
	t := time.Now()
	s := c.DefaultQuery("section", "")
	var section []string
	if s != "" {
		section = strings.Split(s, ",")
	}
	result := sth.Result{}
	info, err := ctrl.client.Info(section...).Result()
	if err == nil {
		parsed := utils.ParseRedisInfo(info)
		for k, v := range parsed {
			result[k] = v
		}
	}
	result["_cost_ms_"] = utils.SinceMS(t)

	return result, err
}
