package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	core "github.com/cfhamlet/os-rq-pod/pod"
	"github.com/gin-gonic/gin"
)

// Resume TODO
func Resume(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return serv.Switch(true)
}

// Pause TODO
func Pause(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return serv.Switch(false)
}

// Info TODO
func Info(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return serv.Info()
}

// RedisInfo TODO
func RedisInfo(c *gin.Context, serv *core.Core) (sth.Result, error) {
	t := time.Now()
	s := c.DefaultQuery("section", "")
	var section []string
	if s != "" {
		section = strings.Split(s, ",")
	}
	info, err := serv.RedisInfo(section...)
	return sth.Result{"info": info, "_cost_ms_": utils.SinceMS(t)}, err
}

// ProcessMemory TODO
func ProcessMemory(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return sth.Result{"memory": serv.ProcessMemory()}, nil
}

// PushRequest TODO
func PushRequest(c *gin.Context, serv *core.Core) (result sth.Result, err error) {
	var req *request.RawRequest = &request.RawRequest{}

	if err = c.ShouldBindJSON(req); err != nil {
		err = InvalidBody(fmt.Sprintf("%s", err))
	} else {
		result, err = serv.QueueBox.PushRequest(req)
	}

	c.Header("Access-Control-Allow-Origin", "*")
	return
}

// PopRequest TODO
func PopRequest(c *gin.Context, serv *core.Core) (result sth.Result, err error) {
	q := c.Query("q")
	var qid sth.QueueID
	qid, err = QueueIDFromQuery(q)
	if err == nil {
		var req *request.Request
		req, err = serv.QueueBox.PopRequest(qid)
		if err == nil {
			c.JSON(http.StatusOK, req)
		}
	}
	return
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
func PauseQueue(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return operateQueueByQuery(c, serv.QueueBox.PauseQueue)
}

// ResumeQueue TODO
func ResumeQueue(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return operateQueueByQuery(c, serv.QueueBox.ResumeQueue)
}

// QueueInfo TODO
func QueueInfo(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return operateQueueByQuery(c, serv.QueueBox.QueueInfo)
}

// DeleteQueue TODO
func DeleteQueue(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return operateQueueByQuery(c, serv.QueueBox.DeleteQueue)
}

// ClearQueue TODO
func ClearQueue(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return operateQueueByQuery(c, serv.QueueBox.ClearQueue)
}

// SyncQueue TODO
func SyncQueue(c *gin.Context, serv *core.Core) (sth.Result, error) {
	return operateQueueByQuery(c, func(qid sth.QueueID) (sth.Result, error) {
		return serv.QueueBox.SyncQueue(qid, true)
	},
	)
}

// ViewQueue  TODO
func ViewQueue(c *gin.Context, serv *core.Core) (result sth.Result, err error) {

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
			return serv.QueueBox.ViewQueue(qid, start, end)
		},
	)
}

// ViewQueues TODO
func ViewQueues(c *gin.Context, serv *core.Core) (result sth.Result, err error) {

	qs := c.DefaultQuery("status", "working")
	status, ok := core.QueueStatusMap[qs]
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
			result = serv.QueueBox.ViewQueues(int(k), int(s), status)
		}
	}

	return
}

// Queues TODO
func Queues(c *gin.Context, serv *core.Core) (result sth.Result, err error) {

	qk := c.DefaultQuery("k", "10")
	k, e := strconv.ParseInt(qk, 10, 64)
	if e != nil {
		err = InvalidQuery(fmt.Sprintf("k=%s %s", qk, err))
	} else if k <= 0 || k > 1000 {
		err = InvalidQuery(fmt.Sprintf("k=%s [1, 1000]", qk))
	}
	if err == nil {
		result = serv.QueueBox.Queues(int(k))
	}
	return
}
