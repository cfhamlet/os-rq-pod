package runner

import (
	"context"
	"net/http"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"go.uber.org/fx"
)

// FailWait TODO
type FailWait chan error

// HTTPServerLifecycle TODO
func HTTPServerLifecycle(lc fx.Lifecycle, server *http.Server, ready Ready) FailWait {
	failWait := make(FailWait)
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				go func() {
					err := <-ready
					if err != nil {
						failWait <- err
					} else {
						log.Logger.Debug("start server", server.Addr)
						var err error
						if err = server.ListenAndServe(); err != http.ErrServerClosed {
							log.Logger.Error("start fail", err)
							failWait <- err
						}
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				log.Logger.Debug("stop server", server.Addr)
				return server.Shutdown(ctx)
			},
		})

	return failWait
}
