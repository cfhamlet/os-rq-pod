package controllers

import (
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/gin-gonic/gin"
)

// ConfigController TODO
type ConfigController struct {
	*serv.Serv
}

// NewConfigController TODO
func NewConfigController(serv *serv.Serv) *ConfigController {
	return &ConfigController{serv}
}

// GetConfig TODO
func (ctrl *ConfigController) GetConfig(c *gin.Context) (result sth.Result, err error) {
	result = sth.Result{}
	conf := ctrl.Conf()
	for _, key := range conf.AllKeys() {
		result[key] = conf.Get(key)
	}
	return
}
