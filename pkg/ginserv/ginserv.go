package ginserv

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/DeanThompson/ginpprof"
	"github.com/cfhamlet/os-rq-pod/pkg/log"
	limits "github.com/gin-contrib/size"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// RouterGroup TODO
type RouterGroup gin.IRouter

// IRoutesHTTPFunc TODO
type IRoutesHTTPFunc func(string, ...gin.HandlerFunc) gin.IRoutes

// NewEngine TODO
func NewEngine(conf *viper.Viper) *gin.Engine {
	debug := false
	if conf.IsSet("debug") {
		debug = conf.GetBool("debug")
	}
	if debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	if debug {
		ginpprof.Wrap(engine)
	}
	return engine
}

// LoadGlobalMiddlewares TODO
func LoadGlobalMiddlewares(conf *viper.Viper, engine *gin.Engine) {
	engine.Use(gin.Recovery())
	if conf.GetBool("http.log.enable") {
		UseLog(engine)
	}
	if conf.IsSet("limit.http.size") {
		limit := conf.GetInt64("limit.http.size")
		engine.Use(limits.RequestSizeLimiter(limit))
	}
}

// NewAPIGroup TODO
func NewAPIGroup(conf *viper.Viper, engine *gin.Engine) RouterGroup {
	return engine.Group(conf.GetString("http.api.path"))
}

// UseLog TODO
func UseLog(engine *gin.Engine) {
	gin.DefaultWriter = os.Stderr
	gin.DisableConsoleColor()
	engine.Use(
		gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[%s] [gin] %s \"%s %s %s\" %d %s %d %d \"%s\" \"%s\"\n",
				param.TimeStamp.Format(log.DefaultLogTimeFormat),
				param.ClientIP,
				param.Method,
				param.Path,
				param.Request.Proto,
				param.StatusCode,
				param.Latency,
				param.Request.ContentLength,
				param.BodySize,
				param.Request.UserAgent(),
				param.ErrorMessage,
			)
		}))
}

// NewServer create HTTP server
func NewServer(conf *viper.Viper, engine *gin.Engine) *http.Server {

	addr := strings.TrimSpace(conf.GetString("http.addr"))

	srv := &http.Server{
		Addr:    addr,
		Handler: engine,
	}
	return srv
}
