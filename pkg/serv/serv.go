package serv

import (
	"context"
	"os"
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/viper"
	messagebus "github.com/vardius/message-bus"
)

// IServ TODO
type IServ interface {
	OnStart(context.Context) error
	OnStop(context.Context) error
}

// Serv TODO
type Serv struct {
	conf    *viper.Viper
	process *process.Process
	status  Status
	message messagebus.MessageBus
	*sync.RWMutex
}

// New TODO
func New(conf *viper.Viper) *Serv {
	proc, err := utils.NewProcess()
	if err != nil {
		panic(err)
	}
	serv := &Serv{
		conf,
		proc,
		Init,
		messagebus.New(1024),
		&sync.RWMutex{},
	}
	return serv
}

// OnStart TODO
func (serv *Serv) OnStart(ctx context.Context) (err error) {
	serv.Lock()
	defer serv.Unlock()
	err = serv.setStatus(Preparing)
	if err == nil {
		err = serv.setStatus(Working)
	}
	switch err.(type) {
	case *StatusConflictError:
		if StopStatus(serv.status) {
			log.Logger.Warning("stop when starting")
			err = nil
		}
	}
	return
}

// OnStop TODO
func (serv *Serv) OnStop(ctx context.Context) (err error) {
	err = serv.SetStatus(Stopping)
	if err == nil {
		err = serv.SetStatus(Stopping)
	}
	return
}

// Status TODO
func (serv *Serv) Status() Status {
	serv.RLock()
	defer serv.RUnlock()
	return serv.status
}

// Process TODO
func (serv *Serv) Process() *process.Process {
	return serv.process
}

// PID TODO
func (serv *Serv) PID() int {
	return os.Getpid()
}

// Conf TODO
func (serv *Serv) Conf() *viper.Viper {
	return serv.conf
}

// setStatus TODO
func (serv *Serv) setStatus(newStatus Status) (err error) {
	oldStatus := serv.status
	if oldStatus == newStatus {
		return
	}
	e := &StatusConflictError{oldStatus, newStatus}
	switch serv.status {
	case Init:
		switch newStatus {
		case Preparing:
		default:
			err = e
		}
	case Preparing:
		switch newStatus {
		case Working:
			fallthrough
		case Paused:
			fallthrough
		case Stopping:
		default:
			err = e
		}
	case Working:
		switch newStatus {
		case Paused:
			fallthrough
		case Stopping:
		default:
			err = e
		}
	case Paused:
		switch newStatus {
		case Working:
			fallthrough
		case Stopping:
		default:
			err = e
		}
	case Stopping:
		switch newStatus {
		case Stopped:
		default:
			err = e
		}
	case Stopped:
		err = e
	}
	if err == nil {
		serv.status = newStatus
	}
	serv.message.Publish(ServStatusChanged, oldStatus, newStatus)
	return

}

// SetStatus TODO
func (serv *Serv) SetStatus(newStatus Status) (err error) {
	serv.Lock()
	defer serv.Unlock()
	return serv.setStatus(newStatus)
}

// DoWithLock TODO
func (serv *Serv) DoWithLock(f func() (interface{}, error), rLock bool) (interface{}, error) {
	if rLock {
		serv.RLock()
		defer serv.RUnlock()
	} else {
		serv.Lock()
		defer serv.Unlock()
	}
	return f()
}

// DoWithLockOnWorkStatus TODO
func (serv *Serv) DoWithLockOnWorkStatus(f func() (interface{}, error), rLock bool, mustWorking bool) (interface{}, error) {
	return serv.DoWithLock(func() (interface{}, error) {
		if !WorkStatus(serv.status) ||
			(mustWorking && serv.status != Working) {
			return nil, &StatusError{serv.status}
		}
		return f()

	}, rLock)
}

// Info TODO
func (serv *Serv) Info() (result sth.Result) {
	return sth.Result{
		"status": serv.status,
		"process": sth.Result{
			"pid":    serv.PID(),
			"memory": utils.MemoryInfo(serv.process),
			"cpu": sth.Result{
				"percent": utils.CPUPercent(serv.process),
			},
		},
	}
}

// Toggle TODO
func (serv *Serv) Toggle(pauseOrResume bool) (sth.Result, error) {
	result, err := serv.DoWithLockOnWorkStatus(
		func() (result interface{}, err error) {
			status := Working
			if !pauseOrResume {
				status = Paused
			}
			err = serv.setStatus(status)
			if err == nil {
				result = serv.Info()
			}
			return
		}, false, false)
	return result.(sth.Result), err
}

// Message TODO
func (serv *Serv) Message() messagebus.MessageBus {
	return serv.message
}
