package runner

import (
	"context"
	"sync"

	"go.uber.org/fx"
)

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
