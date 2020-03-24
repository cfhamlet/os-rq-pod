package runner

import (
	"context"
	"net/http"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"go.uber.org/fx"
)

// HTTPServerLifecycle TODO
func HTTPServerLifecycle(lc fx.Lifecycle, server *http.Server, runner *Runner) {
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				go func() {
					<-runner.WaitReady()
					log.Logger.Debug("start server", server.Addr)
					if err := server.ListenAndServe(); err != http.ErrServerClosed {
						log.Logger.Error("start fail", err)
						runner.Fail(err)
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Logger.Debug("stop server", server.Addr)
				return server.Shutdown(ctx)
			},
		})
}
