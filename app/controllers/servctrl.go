package controllers

import (
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/gin-gonic/gin"
)

// ServController TODO
type ServController struct {
	*serv.Serv
}

// NewServController TODO
func NewServController(serv *serv.Serv) *ServController {
	return &ServController{serv}
}

// Resume TODO
func (ctrl *ServController) Resume(c *gin.Context) (sth.Result, error) {
	return ctrl.Toggle(true)
}

// Pause TODO
func (ctrl *ServController) Pause(c *gin.Context) (sth.Result, error) {
	return ctrl.Toggle(false)
}

// Info TODO
func (ctrl *ServController) Info(c *gin.Context) (sth.Result, error) {
	return ctrl.Serv.Info(), nil
}
