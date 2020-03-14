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
	"github.com/go-redis/redis/v7"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func init() {
	Root.AddCommand(command.NewRunCommand("rq-pod", run))
}

var startFail chan error

func run(conf *viper.Viper) {
	loadConfig := func() (*viper.Viper, error) {
		err := config.LoadConfig(conf, defaultConfig.EnvPrefix, defaultConfig.DefaultConfig)
		return conf, err
	}
	newPod := func(lc fx.Lifecycle, conf *viper.Viper, client *redis.Client) (pod *core.Pod, err error) {
		pod, err = core.NewPod(conf, client)
		if err != nil {
			return
		}
		lc.Append(
			fx.Hook{
				OnStart: func(context.Context) error {
					return pod.OnStart()
				},
				OnStop: func(ctx context.Context) error {
					return pod.OnStop()
				},
			})
		return
	}

	newEngine := func(*core.Pod) *gin.Engine {
		return ginserv.NewEngine(conf)
	}

	app := fx.New(
		fx.Provide(
			loadConfig,
			utils.NewRedisClient,
			newPod,
			newEngine,
			ginserv.NewServer,
			ginserv.NewAPIGroup,
			runner.HTTPServerLifecycle,
		),
		fx.Invoke(
			config.PrintDebugConfig,
			log.ConfigLogging,
			ginserv.LoadGlobalMiddlewares,
			router.InitAPIRouter,
		),
		fx.Populate(&startFail),
	)

	runner.Run(app, startFail)
}
