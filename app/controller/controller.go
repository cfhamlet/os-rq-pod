package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfhamlet/os-go-netloc-rule/netloc"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	core "github.com/cfhamlet/os-rq-pod/pod"
	"github.com/gin-gonic/gin"
)

// Controller TODO
type Controller struct {
	*core.Core
}

// New TODO
func New(serv *core.Core) *Controller {
	return &Controller{serv}
}

// Resume TODO
func (ctrl *Controller) Resume(c *gin.Context) (sth.Result, error) {
	return ctrl.Switch(true)
}

// Pause TODO
func (ctrl *Controller) Pause(c *gin.Context) (sth.Result, error) {
	return ctrl.Switch(false)
}

// Info TODO
func (ctrl *Controller) Info(c *gin.Context) (sth.Result, error) {
	return ctrl.Core.Info()
}

// RedisInfo TODO
func (ctrl *Controller) RedisInfo(c *gin.Context) (sth.Result, error) {
	t := time.Now()
	s := c.DefaultQuery("section", "")
	var section []string
	if s != "" {
		section = strings.Split(s, ",")
	}
	info, err := ctrl.Core.RedisInfo(section...)
	return sth.Result{"info": info, "_cost_ms_": utils.SinceMS(t)}, err
}

// ProcessMemory TODO
func (ctrl *Controller) ProcessMemory(c *gin.Context) (sth.Result, error) {
	return sth.Result{"memory": ctrl.Core.ProcessMemory()}, nil
}

// PushRequest TODO
func (ctrl *Controller) PushRequest(c *gin.Context) (result sth.Result, err error) {
	var req *request.RawRequest = &request.RawRequest{}

	if err = c.ShouldBindJSON(req); err != nil {
		err = InvalidBody(fmt.Sprintf("%s", err))
	} else {
		result, err = ctrl.QueueBox.PushRequest(req)
	}

	c.Header("Access-Control-Allow-Origin", "*")
	return
}

// PopRequest TODO
func (ctrl *Controller) PopRequest(c *gin.Context) (result sth.Result, err error) {
	q := c.Query("q")
	var qid sth.QueueID
	qid, err = QueueIDFromQuery(q)
	if err == nil {
		var req *request.Request
		req, err = ctrl.QueueBox.PopRequest(qid)
		if err == nil {
			c.JSON(http.StatusOK, req)
		}
	}
	return
}

// AddRequestConfig TODO
func (ctrl *Controller) AddRequestConfig(c *gin.Context) (result sth.Result, err error) {
	n := c.Query("netloc")
	var reqConfig *core.RequestConfig
	if err = c.ShouldBindJSON(reqConfig); err != nil {
		err = InvalidBody(fmt.Sprintf("%s", err))
	} else {
		var nlc *netloc.Netloc
		nlc, err = core.NetlocFromString(n)
		if err == nil {
			n, r, err := ctrl.GetExtension("reqwrapper").(*core.RequestWrapper).Add(nlc, reqConfig)
			if err == nil {
				result = sth.Result{
					"replaced": sth.Result{
						"netloc": n,
						"rule":   r,
					},
					"active": sth.Result{
						"netloc": nlc,
						"rule":   reqConfig,
					},
				}
			}
		}
	}
	return
}

// GetRequestConfig TODO
func (ctrl *Controller) GetRequestConfig(c *gin.Context) (result sth.Result, err error) {
	n := c.Query("netloc")
	var nlc *netloc.Netloc
	nlc, err = core.NetlocFromString(n)
	if err != nil {
		n, r := ctrl.GetExtension("reqwrapper").(*core.RequestWrapper).Get(nlc)
		if n == nil {
			err = core.NotExistError(nlc.String())
		} else {
			result = sth.Result{"netloc": n, "rule": r}
		}
	}

	return
}

// MatchRequestConfig TODO
func (ctrl *Controller) MatchRequestConfig(c *gin.Context) (result sth.Result, err error) {
	uri := c.Query("uri")
	if err != nil {
		n, r := ctrl.GetExtension("reqwrapper").(*core.RequestWrapper).MatchURI(uri)
		if n == nil {
			err = core.NotExistError(uri)
		} else {
			result = sth.Result{"netloc": n, "rule": r}
		}
	}

	return
}

// DeleteRequestConfig TODO
func (ctrl *Controller) DeleteRequestConfig(c *gin.Context) (result sth.Result, err error) {
	n := c.Query("netloc")
	var nlc *netloc.Netloc
	nlc, err = core.NetlocFromString(n)
	if err != nil {
		r, e := ctrl.GetExtension("reqwrapper").(*core.RequestWrapper).Delete(nlc)
		if e != nil {
			err = e
		} else {
			if r == nil {
				err = core.NotExistError(nlc.String())
			} else {
				result = sth.Result{"netloc": nlc, "rule": r}
			}
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
func (ctrl *Controller) PauseQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.QueueBox.PauseQueue)
}

// ResumeQueue TODO
func (ctrl *Controller) ResumeQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.QueueBox.ResumeQueue)
}

// QueueInfo TODO
func (ctrl *Controller) QueueInfo(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.QueueBox.QueueInfo)
}

// DeleteQueue TODO
func (ctrl *Controller) DeleteQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.QueueBox.DeleteQueue)
}

// ClearQueue TODO
func (ctrl *Controller) ClearQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, ctrl.QueueBox.ClearQueue)
}

// SyncQueue TODO
func (ctrl *Controller) SyncQueue(c *gin.Context) (sth.Result, error) {
	return operateQueueByQuery(c, func(qid sth.QueueID) (sth.Result, error) {
		return ctrl.QueueBox.SyncQueue(qid, true)
	},
	)
}

// QueuesInfo TODO
func (ctrl *Controller) QueuesInfo(c *gin.Context) (sth.Result, error) {
	return ctrl.QueueBox.Info(), nil
}

// ViewQueue  TODO
func (ctrl *Controller) ViewQueue(c *gin.Context) (result sth.Result, err error) {

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
			return ctrl.QueueBox.ViewQueue(qid, start, end)
		},
	)
}

// ViewQueues TODO
func (ctrl *Controller) ViewQueues(c *gin.Context) (result sth.Result, err error) {

	qs := c.DefaultQuery("status", utils.Text(core.Working))
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
			result = ctrl.QueueBox.ViewQueues(int(k), int(s), status)
		}
	}

	return
}

// Queues TODO
func (ctrl *Controller) Queues(c *gin.Context) (result sth.Result, err error) {

	qk := c.DefaultQuery("k", "10")
	k, e := strconv.ParseInt(qk, 10, 64)
	if e != nil {
		err = InvalidQuery(fmt.Sprintf("k=%s %s", qk, err))
	} else if k <= 0 || k > 1000 {
		err = InvalidQuery(fmt.Sprintf("k=%s [1, 1000]", qk))
	}
	if err == nil {
		result = ctrl.QueueBox.Queues(int(k))
	}
	return
}
