package controllers

import (
	"fmt"
	"net/http"

	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pod/queuebox"
	"github.com/cfhamlet/os-rq-pod/pod/redisguard"
	"github.com/gin-gonic/gin"
)

// RequestController TODO
type RequestController struct {
	queueBox *queuebox.QueueBox
	guard    *redisguard.Guard
}

// NewRequestController TODO
func NewRequestController(queueBox *queuebox.QueueBox, guard *redisguard.Guard) *RequestController {
	return &RequestController{queueBox, guard}
}

// EnqueueRequest TODO
func (ctrl *RequestController) EnqueueRequest(c *gin.Context) (result sth.Result, err error) {
	if err = ctrl.guard.Guard(); err != nil {
		return
	}
	var raw *request.RawRequest = &request.RawRequest{}

	if err = c.ShouldBindJSON(raw); err != nil {
		err = InvalidBody(fmt.Sprintf("%s", err))
		return
	}
	var req *request.Request
	req, err = request.NewRequest(raw)
	if err != nil {
		return
	}
	result, err = ctrl.queueBox.EnqueueRequest(req, false)
	c.Header("Access-Control-Allow-Origin", "*")
	return
}

// DequeueRequest TODO
func (ctrl *RequestController) DequeueRequest(c *gin.Context) (result sth.Result, err error) {
	q := c.Query("q")
	var qid sth.QueueID
	qid, err = QueueIDFromQuery(q)
	if err == nil {
		var req *request.Request
		req, err = ctrl.queueBox.DequeueRequest(qid)
		if err == nil {
			c.JSON(http.StatusOK, req)
		}
	}
	return
}
