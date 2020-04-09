package controllers

import (
	"fmt"
	"strconv"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/cfhamlet/os-rq-pod/pod/queuebox"
	"github.com/gin-gonic/gin"
)

// QueuesController TODO
type QueuesController struct {
	queueBox *queuebox.QueueBox
}

// NewQueuesController TODO
func NewQueuesController(queueBox *queuebox.QueueBox) *QueuesController {
	return &QueuesController{queueBox}
}

// QueuesInfo TODO
func (ctrl *QueuesController) QueuesInfo(c *gin.Context) (sth.Result, error) {
	return ctrl.queueBox.Info(), nil
}

// ViewQueues TODO
func (ctrl *QueuesController) ViewQueues(c *gin.Context) (result sth.Result, err error) {

	qs := c.DefaultQuery("status", utils.Text(queuebox.Working))
	status, ok := queuebox.QueueStatusMap[qs]
	if !ok {
		err = InvalidQuery(fmt.Sprintf(`invalid status '%s'`, qs))
		return
	}

	qk := c.DefaultQuery("k", "10")
	k, e := strconv.ParseInt(qk, 10, 64)
	if e != nil {
		err = InvalidQuery(fmt.Sprintf("k=%s %s", qk, err))
	} else if k <= 0 || k > 1000 {
		err = InvalidQuery(fmt.Sprintf("k=%s [1, 1000]", qk))
	}

	if err == nil {
		qs := c.DefaultQuery("start", "0")
		s, e := strconv.ParseInt(qs, 10, 64)
		if e != nil {
			err = InvalidQuery(fmt.Sprintf("start=%s %s", qs, err))
		} else {
			result = ctrl.queueBox.ViewQueues(int(k), int(s), status)
		}
	}

	return
}

// Queues TODO
func (ctrl *QueuesController) Queues(c *gin.Context) (result sth.Result, err error) {

	qk := c.DefaultQuery("k", "10")
	k, e := strconv.ParseInt(qk, 10, 64)
	if e != nil {
		err = InvalidQuery(fmt.Sprintf("k=%s %s", qk, err))
	} else if k <= 0 || k > 1000 {
		err = InvalidQuery(fmt.Sprintf("k=%s [1, 1000]", qk))
	}
	if err == nil {
		result = ctrl.queueBox.Queues(int(k))
	}
	return
}
