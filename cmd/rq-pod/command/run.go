package command

import (
	"context"

	"github.com/cfhamlet/os-rq-pod/app/router"
	defaultConfig "github.com/cfhamlet/os-rq-pod/internal/config"
	"github.com/cfhamlet/os-rq-pod/pkg/command"
	"github.com/cfhamlet/os-rq-pod/pkg/config"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/runner"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	core "github.com/cfhamlet/os-rq-pod/pod"
	"github.com/gin-gonic/gin"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func init() {
	Root.AddCommand(command.NewRunCommand("rq-pod", run))
}

// PodLifecycle TODO
func PodLifecycle(lc fx.Lifecycle, pod *core.Pod) runner.Ready {
	ready := make(runner.Ready)
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				go func() {
					err := pod.OnStart()
					ready <- err
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return pod.OnStop()
			},
		})
	return ready

}

func run(conf *viper.Viper) {
	newConfig := func() (*viper.Viper, error) {
		err := config.LoadConfig(conf, defaultConfig.EnvPrefix, defaultConfig.DefaultConfig)
		return conf, err
	}

	newEngine := func(*core.Pod) *gin.Engine {
		return ginserv.NewEngine(conf)
	}

	var failWait runner.FailWait

	app := fx.New(
		fx.Provide(
			newConfig,
			utils.NewRedisClient,
			core.NewPod,
			newEngine,
			ginserv.NewServer,
			ginserv.NewAPIGroup,
			PodLifecycle,
			runner.HTTPServerLifecycle,
		),
		fx.Invoke(
			config.PrintDebugConfig,
			log.ConfigLogging,
			ginserv.LoadGlobalMiddlewares,
			router.InitAPIRouter,
		),
		fx.Populate(&failWait),
	)

	runner.Run(app, failWait)
}
