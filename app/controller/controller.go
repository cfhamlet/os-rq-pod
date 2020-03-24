package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	core "github.com/cfhamlet/os-rq-pod/pod"
	"github.com/gin-gonic/gin"
)

// QueueIDFromQuery TODO
func QueueIDFromQuery(q string) (qid core.QueueID, err error) {
	if strings.Contains(q, "://") {
		p, err := utils.NewParsedURL(q)
		if err == nil {
			qid = core.CreateQueueID(p.Host, p.Port, p.Parsed.Scheme)
		}
	} else {
		s := strings.Split(q, ":")
		switch len(s) {
		case 1:
			qid = core.CreateQueueID(q, "", "http")
		case 2:
			qid = core.CreateQueueID(s[0], s[1], "http")
		case 3: // QueueID
			qid = core.CreateQueueID(s[0], s[1], s[2])
		default:
			err = InvalidQuery(fmt.Sprintf("%q", q))
		}
	}

	return
}

// AddRequest TODO
func AddRequest(c *gin.Context, pod *core.Pod) (result core.Result, err error) {
	var req *request.RawRequest = &request.RawRequest{}

	if err = c.ShouldBindJSON(req); err != nil {
		err = InvalidBody(fmt.Sprintf("%s", err))
	} else {
		result, err = pod.AddRequest(req)
	}

	c.Header("Access-Control-Allow-Origin", "*")
	return
}

// CallByQueueID TODO
type CallByQueueID func(core.QueueID) (core.Result, error)

// Resume TODO
func Resume(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return pod.Resume()
}

// Pause TODO
func Pause(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return pod.Pause()
}

// Info TODO
func Info(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return pod.Info()
}

// GetRequest TODO
func GetRequest(c *gin.Context, pod *core.Pod) (result core.Result, err error) {
	q := c.Query("q")
	var qid core.QueueID
	qid, err = QueueIDFromQuery(q)
	if err == nil {
		var req *request.Request
		req, err = pod.GetRequest(qid)
		if err == nil {
			c.JSON(http.StatusOK, req)
		}
	}
	return
}

func operateQueueByQuery(c *gin.Context, f CallByQueueID) (result core.Result, err error) {
	q := c.Query("q")

	qid, err := QueueIDFromQuery(q)
	if err == nil {
		result, err = f(qid)
	}
	if result == nil {
		result = core.Result{"qid": qid}
	}
	return
}

// PauseQueue TODO
func PauseQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return operateQueueByQuery(c, pod.PauseQueue)
}

// ResumeQueue TODO
func ResumeQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return operateQueueByQuery(c, pod.ResumeQueue)
}

// QueueInfo TODO
func QueueInfo(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return operateQueueByQuery(c, pod.QueueInfo)
}

// DeleteQueue TODO
func DeleteQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return operateQueueByQuery(c, pod.DeleteQueue)
}

// ClearQueue TODO
func ClearQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return operateQueueByQuery(c, pod.ClearQueue)
}

// SyncQueue TODO
func SyncQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return operateQueueByQuery(c, pod.ForceSyncQueue)
}

// ViewQueue  TODO
func ViewQueue(c *gin.Context, pod *core.Pod) (result core.Result, err error) {

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
		func(qid core.QueueID) (core.Result, error) {
			return pod.ViewQueue(qid, start, end)
		},
	)
}

func infoResult(info interface{}, err error) (core.Result, error) {
	return core.Result{"info": info}, err
}

// RedisMemory TODO
func RedisMemory(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return infoResult(pod.Client.Info("memory").Result())
}

// RedisInfo TODO
func RedisInfo(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return infoResult(pod.Client.Info().Result())
}

// ProcessMemory TODO
func ProcessMemory(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return infoResult(pod.Process.MemoryInfo())
}

// ViewQueues TODO
func ViewQueues(c *gin.Context, pod *core.Pod) (result core.Result, err error) {

	qs := c.DefaultQuery("status", utils.Text(core.QueueWorking))
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
			result, err = pod.ViewQueues(int(k), int(s), status)
		}
	}

	return
}

// Queues TODO
func Queues(c *gin.Context, pod *core.Pod) (result core.Result, err error) {

	qk := c.DefaultQuery("k", "10")
	k, e := strconv.ParseInt(qk, 10, 64)
	if e != nil {
		err = InvalidQuery(fmt.Sprintf("k=%s %s", qk, err))
	} else if k <= 0 || k > 1000 {
		err = InvalidQuery(fmt.Sprintf("k=%s [1, 1000]", qk))
	}
	if err == nil {
		result, err = pod.Queues(int(k))
	}
	return
}
