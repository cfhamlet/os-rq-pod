package command

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/app/routers"
	"github.com/cfhamlet/os-rq-pod/pkg/command"
	"github.com/cfhamlet/os-rq-pod/pkg/config"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	"github.com/cfhamlet/os-rq-pod/pkg/runner"
	"github.com/cfhamlet/os-rq-pod/pkg/serv"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/cfhamlet/os-rq-pod/pod/global"
	"github.com/cfhamlet/os-rq-pod/pod/queuebox"
	"github.com/cfhamlet/os-rq-pod/pod/redisconfig"
	"github.com/cfhamlet/os-rq-pod/pod/redisguard"
	"github.com/cfhamlet/os-rq-pod/pod/reqwrap"

	"github.com/go-redis/redis/v7"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func init() {
	Root.AddCommand(command.NewRunCommand("rq-pod", run))
}

func run(conf *viper.Viper) {
	newConfig := func() (*viper.Viper, error) {
		err := config.LoadConfig(conf, global.EnvPrefix, global.DefaultConfig)
		return conf, err
	}

	newRdsConfig := func(serv *serv.Serv, client *redis.Client) *redisconfig.RedisConfig {
		return redisconfig.New(serv, client,
			[]string{
				"limit.queue.length",
				"limit.queue.num",
				"limit.redis.memory",
				"limit.request.num",
			},
		)
	}

	servGo := func(lc fx.Lifecycle, serv *serv.Serv, r *runner.Runner) {
		runner.ServWait(lc, serv, r)
	}

	gboxGo := func(lc fx.Lifecycle, queueBox *queuebox.QueueBox, r *runner.Runner) {
		runner.ServGo(lc, queueBox, r)
	}

	reqwGo := func(lc fx.Lifecycle, reqw *reqwrap.RequestWrapper, r *runner.Runner) {
		runner.ServGo(lc, reqw, r)
	}

	confGo := func(lc fx.Lifecycle, rds *redisconfig.RedisConfig, r *runner.Runner) {
		runner.ServGo(lc, rds, r)
	}

	guardGo := func(lc fx.Lifecycle, guard *redisguard.Guard, r *runner.Runner) {
		runner.ServWait(lc, guard, r)
	}

	var r *runner.Runner

	app := fx.New(
		fx.Provide(
			runner.New,
			newConfig,
			serv.New,
			queuebox.New,
			redisguard.New,
			newRdsConfig,
			utils.NewRedisClient,
			reqwrap.New,
			ginserv.NewEngine,
			ginserv.NewServer,
			ginserv.NewAPIGroup,
			controllers.NewConfigController,
			controllers.NewQueueController,
			controllers.NewQueuesController,
			controllers.NewRedisController,
			controllers.NewRequestWrapperController,
			controllers.NewRequestController,
			controllers.NewServController,
		),
		fx.Invoke(
			config.PrintDebugConfig,
			log.ConfigLogging,
			ginserv.LoadGlobalMiddlewares,
			confGo,
			guardGo,
			gboxGo,
			reqwGo,
			servGo,
			routers.RouteRedisCtrl,
			routers.RouteQueueCtrl,
			routers.RouteQueuesCtrl,
			routers.RouteServCtrl,
			routers.RouteRequestWrapperCtrl,
			routers.RouteConfigCtrl,
			routers.RouteRequestCtrl,
			runner.HTTPServerLifecycle,
		),
		fx.Populate(&r),
	)

	r.Run(app)
}
