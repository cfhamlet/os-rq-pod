package runner

import (
	"context"
	"sync"

	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	messagebus "github.com/vardius/message-bus"
	"go.uber.org/fx"
)

// Runner TODO
type Runner struct {
	messagebus.MessageBus
	ready *sync.WaitGroup
	fail  chan error
}

// New TODO
func New() *Runner {
	return &Runner{
		messagebus.New(100),
		nil,
		make(chan error),
	}
}

// WaitReady TODO
func (runner *Runner) WaitReady() {
	if runner.ready != nil {
		runner.ready.Wait()
	}
}

// AddReady TODO
func (runner *Runner) AddReady() *sync.WaitGroup {
	if runner.ready == nil {
		runner.ready = &sync.WaitGroup{}
	}
	runner.ready.Add(1)
	return runner.ready
}

// PopReady TODO
func (runner *Runner) PopReady() *sync.WaitGroup {
	if runner.ready == nil {
		return nil
	}
	ready := runner.ready
	runner.ready = &sync.WaitGroup{}
	return ready
}

// Fail TODO
func (runner *Runner) Fail(err error) {
	runner.fail <- err
}

// WaitFail TODO
func (runner *Runner) WaitFail() chan error {
	return runner.fail
}

// ServWait TODO
func ServWait(lc fx.Lifecycle, serv serv.IServ, runner *Runner) {
	servLifecycle(lc, serv, runner, true)
}

// ServGo TODO
func ServGo(lc fx.Lifecycle, serv serv.IServ, runner *Runner) {
	servLifecycle(lc, serv, runner, false)
}

func servLifecycle(lc fx.Lifecycle, serv serv.IServ, runner *Runner, waitReady bool) {
	lc.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				var wait *sync.WaitGroup
				if waitReady {
					wait = runner.PopReady()
				}
				ready := runner.AddReady()
				go func() {
					if wait != nil {
						wait.Wait()
					}
					err := serv.OnStart(ctx)
					if err == nil {
						ready.Done()
					} else {
						runner.Fail(err)
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return serv.OnStop(ctx)
			},
		})

}

// Run TODO
func (runner *Runner) Run(app *fx.App) {
	defer func() {
		if err := app.Stop(context.Background()); err != nil {
			panic(err)
		}
	}()
	if err := app.Start(context.Background()); err != nil {
		panic(err)
	}
	select {
	case <-runner.WaitFail():
	case <-app.Done():
	}
}
