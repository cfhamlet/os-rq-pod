package runner

import (
	"context"
	"net/http"

	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"go.uber.org/fx"
)

// HTTPServ TODO
type HTTPServ struct {
	*http.Server
}

// OnStart TODO
func (serv *HTTPServ) OnStart(ctx context.Context) error {
	log.Logger.Debug("start server", serv.Addr)
	if err := serv.ListenAndServe(); err != http.ErrServerClosed {
		log.Logger.Error("start fail", err)
		return err
	}
	return nil
}

// OnStop TODO
func (serv *HTTPServ) OnStop(ctx context.Context) error {
	log.Logger.Debug("stop server", serv.Addr)
	return serv.Shutdown(ctx)
}

// HTTPServerLifecycle TODO
func HTTPServerLifecycle(lc fx.Lifecycle, server *http.Server, runner *Runner) {
	serv := &HTTPServ{server}
	ServWait(lc, serv, runner)
}
