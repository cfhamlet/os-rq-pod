package routers

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv/route"
)

// RouteRedisCtrl TODO
func RouteRedisCtrl(root ginserv.RouterGroup, ctrl *controllers.RedisController) {
	g := root.Group("/system/info/")
	routes := []*route.Route{
		route.New(g.GET, "/redis/", ctrl.RedisInfo),
	}
	route.Bind(routes, controllers.ErrorCode)
}
