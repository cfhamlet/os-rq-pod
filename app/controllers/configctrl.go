package controllers

import (
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pod/redisconfig"
	"github.com/gin-gonic/gin"
)

// ConfigController TODO
type ConfigController struct {
	rdsconfig *redisconfig.RedisConfig
}

// NewConfigController TODO
func NewConfigController(rdsconfig *redisconfig.RedisConfig) *ConfigController {
	return &ConfigController{rdsconfig}
}

// SetConfig TODO
func (ctrl *ConfigController) SetConfig(c *gin.Context) (result sth.Result, err error) {
	kvs := map[string]interface{}{}
	if err = c.ShouldBindJSON(&kvs); err != nil {
		err = InvalidBody(err.Error())
		return
	}
	result, err = ctrl.rdsconfig.SetConfigs(kvs)
	return
}

// GetConfig TODO
func (ctrl *ConfigController) GetConfig(c *gin.Context) (result sth.Result, err error) {
	key := c.Query("key")
	if key == "" {
		result = ctrl.rdsconfig.AllConfigs()
	} else {
		var value interface{}
		value, err = ctrl.rdsconfig.GetConfig(key)
		if err == nil {
			result = sth.Result{key: value}
		}
	}

	return
}
