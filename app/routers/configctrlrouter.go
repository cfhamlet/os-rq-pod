package routers

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv/route"
)

// RouteConfigCtrl TODO
func RouteConfigCtrl(root ginserv.RouterGroup, ctrl *controllers.ConfigController) {
	g := root.Group("/system/config/")
	routes := []*route.Route{
		route.New(g.GET, "/", ctrl.GetConfig),
	}
	route.Bind(routes, controllers.ErrorCode)
}
