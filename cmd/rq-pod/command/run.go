package command

import (
	"github.com/cfhamlet/os-rq-pod/app/router"
	defaultConfig "github.com/cfhamlet/os-rq-pod/internal/config"
	"github.com/cfhamlet/os-rq-pod/pkg/command"
	"github.com/cfhamlet/os-rq-pod/pkg/config"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/runner"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	core "github.com/cfhamlet/os-rq-pod/pod"

	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func init() {
	Root.AddCommand(command.NewRunCommand("rq-pod", run))
}

func run(conf *viper.Viper) {
	newConfig := func() (*viper.Viper, error) {
		err := config.LoadConfig(conf, defaultConfig.EnvPrefix, defaultConfig.DefaultConfig)
		return conf, err
	}

	podLifecycle := func(lc fx.Lifecycle, serv *core.Core, r *runner.Runner) {
		runner.ServeFlowLifecycle(lc, serv, r)
	}

	var r *runner.Runner

	app := fx.New(
		fx.Provide(
			runner.New,
			newConfig,
			utils.NewRedisClient,
			core.NewPod,
			ginserv.NewEngine,
			ginserv.NewServer,
			ginserv.NewAPIGroup,
		),
		fx.Invoke(
			config.PrintDebugConfig,
			log.ConfigLogging,
			ginserv.LoadGlobalMiddlewares,
			podLifecycle,
			router.InitAPIRouter,
			runner.HTTPServerLifecycle,
		),
		fx.Populate(&r),
	)

	r.Run(app)
}
