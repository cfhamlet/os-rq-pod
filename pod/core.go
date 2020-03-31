package pod

import (
	"time"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
)

// Core TODO
type Core struct {
	*serv.RedisServ
	QueueBox *QueueBox
}

// NewPod creates RQ object
func NewPod(conf *viper.Viper, client *redis.Client) (core *Core) {

	core = &Core{
		serv.NewRedisServ(conf, client),
		nil,
	}

	queueBox := NewQueueBox(core)
	core.QueueBox = queueBox

	return
}

// OnStart TODO
func (core *Core) OnStart() (err error) {
	err = core.SetStatus(serv.Preparing, true)
	if err == nil {
		err = core.QueueBox.Load()
		if err == nil {
			err = core.SetStatus(serv.Working, true)
		}
	}

	switch err.(type) {
	case *serv.StatusChangeError:
		if serv.StopStatus(core.Status(true)) {
			log.Logger.Warning("stop when starting")
			return nil
		}
	}
	return
}

// OnStop TODO
func (core *Core) OnStop() error {
	_, err := core.DoWithLock(
		func() (interface{}, error) {
			err := core.SetStatus(serv.Stopping, false)
			if err == nil {
				err = core.SetStatus(serv.Stopped, false)
			}
			return nil, err
		}, false)
	return err
}

// metaInfo TODO
func (core *Core) metaInfo() (result sth.Result) {
	return sth.Result{
		"stats":  core.QueueBox.Info(),
		"status": core.Status(false),
		"process": sth.Result{
			"pid":    core.Getpid(),
			"memory": core.ProcessMemory(),
			"cpu": sth.Result{
				"percent": core.CPUPercent(),
			},
		},
	}
}

// Info TODO
func (core *Core) Info() (sth.Result, error) {
	result, err := core.DoWithLock(
		func() (interface{}, error) {
			result := core.metaInfo()
			t := time.Now()
			r, err := core.RedisInfo()
			rinfo := sth.Result{
				"info":      r,
				"_cost_ms_": utils.SinceMS(t),
			}
			result["redis"] = rinfo
			return result, err

		}, true)
	return result.(sth.Result), err
}

// Switch TODO
func (core *Core) Switch(pauseOrResume bool) (sth.Result, error) {
	result, err := core.DoWithLockOnWorkStatus(
		func() (result interface{}, err error) {
			status := serv.Working
			if !pauseOrResume {
				status = serv.Paused
			}
			err = core.SetStatus(status, false)
			if err == nil {
				result = core.metaInfo()
			}
			return
		}, false, false)
	return result.(sth.Result), err
}
