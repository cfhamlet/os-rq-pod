package routers

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv/route"
)

// RouteServCtrl TODO
func RouteServCtrl(root ginserv.RouterGroup, ctrl *controllers.ServController) {
	g := root.Group("/system/")
	routes := []*route.Route{
		route.New(g.POST, "/ctrl/pause/", ctrl.Pause),
		route.New(g.POST, "/ctrl/resume/", ctrl.Resume),
		route.New(g.GET, "/info/serv/", ctrl.Info),
	}
	route.Bind(routes, controllers.ErrorCode)
}
