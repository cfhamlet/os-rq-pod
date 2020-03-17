package runner

import (
	"context"
	"sync"

	"go.uber.org/fx"
)

// Ready TODO
type Ready chan error

// ServeFlow TODO
type ServeFlow interface {
	OnStart() error
	OnStop() error
}

// ServeFlowLifecycle TODO
func ServeFlowLifecycle(lc fx.Lifecycle, serv ServeFlow) Ready {
	ready := make(Ready)
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				go func() {
					err := serv.OnStart()
					ready <- err
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return serv.OnStop()
			},
		})
	return ready

}

func merge(cs ...<-chan error) <-chan error {
	out := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan error) {
			for v := range c {
				out <- v
			}
			wg.Done()
		}(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Run TODO
func Run(app *fx.App, noWaits ...<-chan error) {
	defer func() {
		if err := app.Stop(context.Background()); err != nil {
			panic(err)
		}
	}()
	if err := app.Start(context.Background()); err != nil {
		panic(err)
	}
	select {
	case <-merge(noWaits...):
	case <-app.Done():
	}
}
