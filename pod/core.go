package pod

import (
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
)

// Core TODO
type Core struct {
	*serv.RedisServ
	QueueBox *QueueBox
}

// New creates RQ object
func New(conf *viper.Viper, client *redis.Client) (core *Core) {

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

// Info TODO
func (core *Core) Info() (sth.Result, error) {
	result, err := core.DoWithLock(
		func() (interface{}, error) {
			result, err := core.MetaInfo()
			result["stats"] = core.QueueBox.Info()
			return result, err
		}, true)
	return result.(sth.Result), err
}
