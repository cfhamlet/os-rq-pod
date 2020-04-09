package controllers

import (
	"fmt"
	"strconv"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/cfhamlet/os-rq-pod/pod/queuebox"
	"github.com/gin-gonic/gin"
)

// QueueController TODO
type QueueController struct {
	queueBox *queuebox.QueueBox
}

// NewQueueController TODO
func NewQueueController(queueBox *queuebox.QueueBox) *QueueController {
	return &QueueController{queueBox}
}

func operateQueueByQuery(c *gin.Context, f CallByQueueID) (result sth.Result, err error) {
	q := c.Query("q")

	qid, err := QueueIDFromQuery(q)
	if err == nil {
		result, err = f(qid)
	}
	if result == nil {
		result = sth.Result{"qid": qid}
	}
	return
}

// PauseQueue TODO
func (ctrl *QueueController) PauseQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.queueBox.PauseQueue)
}

// ResumeQueue TODO
func (ctrl *QueueController) ResumeQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.queueBox.ResumeQueue)
}

// QueueInfo TODO
func (ctrl *QueueController) QueueInfo(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.queueBox.QueueInfo)
}

// DeleteQueue TODO
func (ctrl *QueueController) DeleteQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.queueBox.DeleteQueue)
}

// ClearQueue TODO
func (ctrl *QueueController) ClearQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.queueBox.ClearQueue)
}

// SyncQueue TODO
func (ctrl *QueueController) SyncQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, func(qid sth.QueueID) (sth.Result, error) {
		return ctrl.queueBox.SyncQueue(qid, true)
	},
	)
}

// ViewQueue  TODO
func (ctrl *QueueController) ViewQueue(c *gin.Context) (result sth.Result, err error) {

	s := c.DefaultQuery("s", "0")
	e := c.DefaultQuery("e", "0")
	var start, end int64

	start, err = strconv.ParseInt(s, 10, 64)
	if err == nil {
		end, err = strconv.ParseInt(e, 10, 64)
		if err == nil {
			if start*end < 0 || utils.AbsInt64(start-end) >= 100 {
				err = InvalidQuery(fmt.Sprintf("range s=%s e=%s", s, e))
			}
		}
	}
	if err != nil {
		return
	}

	return operateQueueByQuery(c,
		func(qid sth.QueueID) (sth.Result, error) {
			return ctrl.queueBox.ViewQueue(qid, start, end)
		},
	)
}
