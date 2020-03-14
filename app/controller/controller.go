package controller

import (
	"fmt"
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
func GetRequest(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return callOperateQueueByQuery(c, pod, pod.GetRequest)
}

// OperateQueueByQuery TODO
func operateQueueByQuery(c *gin.Context, opt core.QueueOperation) (result core.Result, err error) {
	q := c.Query("q")

	qid, err := QueueIDFromQuery(q)
	if err == nil {
		result, err = opt.Operate(qid)
	}
	if result == nil {
		result = core.Result{"qid": qid}
	}
	return
}

func callOperateQueueByQuery(c *gin.Context, pod *core.Pod, f core.FuncWithQueueID) (core.Result, error) {
	return operateQueueByQuery(c, core.NewCallByQueueIDOperation(pod, f))
}

// PauseQueue TODO
func PauseQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return callOperateQueueByQuery(c, pod, pod.PauseQueue)
}

// ResumeQueue TODO
func ResumeQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return callOperateQueueByQuery(c, pod, pod.ResumeQueue)
}

// QueueInfo TODO
func QueueInfo(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return callOperateQueueByQuery(c, pod, pod.QueueInfo)
}

// DeleteQueue TODO
func DeleteQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return callOperateQueueByQuery(c, pod, pod.DeleteQueue)
}

// ClearQueue TODO
func ClearQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return callOperateQueueByQuery(c, pod, pod.ClearQueue)
}

// SyncQueue TODO
func SyncQueue(c *gin.Context, pod *core.Pod) (core.Result, error) {
	return callOperateQueueByQuery(c, pod, pod.ForceSyncQueue)
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
	if err == nil {
		result, err = operateQueueByQuery(c,
			core.NewViewQueueOperation(pod, start, end))
	}

	return
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

// Queues TODO
func Queues(c *gin.Context, pod *core.Pod) (result core.Result, err error) {

	qt := c.DefaultQuery("t", "w")
	var status core.QueueStatus
	switch qt {
	case "w":
		status = core.QueueWorking
	case "p":
		status = core.QueuePaused
	default:
		err = InvalidQuery(fmt.Sprint(`t ["w", "p"]`))
		return
	}

	qr := c.DefaultQuery("r", "1")
	random, e := strconv.ParseBool(qr)
	if e != nil {
		err = InvalidQuery(fmt.Sprintf("r=%s %s", qr, err))
		return
	}
	qk := c.DefaultQuery("k", "10")
	k, e := strconv.ParseInt(qk, 10, 64)
	if e != nil {
		err = InvalidQuery(fmt.Sprintf("k=%s %s", qk, err))
	} else if k <= 0 || k > 1000 {
		err = InvalidQuery(fmt.Sprintf("k=%s [1, 1000]", qk))
	}
	if err != nil {
		return
	}

	if random {
		result = pod.RandomQueues(int(k), status)
	} else {
		qs := c.DefaultQuery("s", "0")
		s, e := strconv.ParseInt(qs, 10, 64)
		if e != nil {
			err = InvalidQuery(fmt.Sprintf("s=%s %s", qs, err))
		}
		result = pod.OrderedQueues(int(k), int(s), status)
	}
	return
}
