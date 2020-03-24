package runner

import (
	"context"

	messagebus "github.com/vardius/message-bus"
	"go.uber.org/fx"
)

// ServeFlow TODO
type ServeFlow interface {
	OnStart() error
	OnStop() error
}

// Runner TODO
type Runner struct {
	messagebus.MessageBus
	ready chan struct{}
	fail  chan error
}

// New TODO
func New() *Runner {
	return &Runner{
		messagebus.New(100),
		make(chan struct{}),
		make(chan error),
	}
}

// WaitReady TODO
func (runner *Runner) WaitReady() chan struct{} {
	return runner.ready
}

// Ready TODO
func (runner *Runner) Ready() {
	runner.ready <- struct{}{}
}

// Fail TODO
func (runner *Runner) Fail(err error) {
	runner.fail <- err
}

// WaitFail TODO
func (runner *Runner) WaitFail() chan error {
	return runner.fail
}

// ServeFlowLifecycle TODO
func ServeFlowLifecycle(lc fx.Lifecycle, serv ServeFlow, runner *Runner) {
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				go func() {
					err := serv.OnStart()
					if err == nil {
						runner.Ready()
					} else {
						runner.Fail(err)
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return serv.OnStop()
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
