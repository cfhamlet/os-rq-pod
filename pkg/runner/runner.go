package runner

import (
	"context"
	"net/http"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"go.uber.org/fx"
)

// HTTPServerLifecycle TODO
func HTTPServerLifecycle(lc fx.Lifecycle, server *http.Server) chan error {
	startFail := make(chan error, 1)
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				go func() {
					log.Logger.Debug("start server", server.Addr)
					if err := server.ListenAndServe(); err != http.ErrServerClosed {
						log.Logger.Error("start fail", err)
						startFail <- err
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Logger.Debug("stop server", server.Addr)
				return server.Shutdown(ctx)
			},
		})

	return startFail
}

// Run TODO
func Run(app *fx.App, startFail chan error) {
	defer func() {
		if err := app.Stop(context.Background()); err != nil {
			panic(err)
		}
	}()
	if err := app.Start(context.Background()); err != nil {
		panic(err)
	}
	select {
	case <-startFail:
	case <-app.Done():
	}
}
