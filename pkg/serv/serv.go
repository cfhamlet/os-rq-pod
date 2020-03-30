package serv

import (
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/shirou/gopsutil/process"
	"github.com/spf13/viper"
)

// Serv TODO
type Serv struct {
	conf    *viper.Viper
	process *process.Process
	status  Status
	*sync.RWMutex
}

// New TODO
func New(conf *viper.Viper) *Serv {
	proc, err := utils.NewProcess()
	if err != nil {
		panic(err)
	}
	return &Serv{
		conf,
		proc,
		Init,
		&sync.RWMutex{},
	}
}

// Status TODO
func (serv *Serv) Status(lock bool) Status {
	if lock {
		serv.RLock()
		defer serv.RUnlock()
	}
	return serv.status
}

// ProcessMemory TODO
func (serv *Serv) ProcessMemory() *process.MemoryInfoStat {
	mem, err := serv.process.MemoryInfo()
	if err != nil {
		return nil
	}
	return mem
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
	e := &StatusTransError{oldStatus, newStatus}
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
	return

}

// SetStatus TODO
func (serv *Serv) SetStatus(newStatus Status, lock bool) (err error) {
	if !lock {
		return serv.setStatus(newStatus)
	}
	_, err = serv.DoWithLock(
		func() (interface{}, error) {
			return nil, serv.setStatus(newStatus)
		},
		false)
	return
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
func (serv *Serv) DoWithLockOnWorkStatus(f func() (interface{}, error), rLock bool) (interface{}, error) {
	return serv.DoWithLock(func() (interface{}, error) {
		if !WorkStatus(serv.Status(false)) {
			return nil, &StatusError{serv.Status(false)}
		}
		return f()

	},
		rLock)
}