package controllers

import (
	"fmt"
	"net/http"

	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pod/queuebox"
	"github.com/gin-gonic/gin"
)

// RequestController TODO
type RequestController struct {
	queueBox *queuebox.QueueBox
}

// NewRequestController TODO
func NewRequestController(queueBox *queuebox.QueueBox) *RequestController {
	return &RequestController{queueBox}
}

// PushRequest TODO
func (ctrl *RequestController) PushRequest(c *gin.Context) (result sth.Result, err error) {
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
	result, err = ctrl.queueBox.PushRequest(req)
	c.Header("Access-Control-Allow-Origin", "*")
	return
}

// PopRequest TODO
func (ctrl *RequestController) PopRequest(c *gin.Context) (result sth.Result, err error) {
	q := c.Query("q")
	var qid sth.QueueID
	qid, err = QueueIDFromQuery(q)
	if err == nil {
		var req *request.Request
		req, err = ctrl.queueBox.PopRequest(qid)
		if err == nil {
			c.JSON(http.StatusOK, req)
		}
	}
	return
}
