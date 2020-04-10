package routers

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv/route"
)

// RouteServCtrl TODO
func RouteServCtrl(root ginserv.RouterGroup, ctrl *controllers.ServController) {
	g := root.Group("/serv/")
	routes := []*route.Route{
		route.New(g.POST, "/pause/", ctrl.Pause),
		route.New(g.POST, "/resume/", ctrl.Resume),
		route.New(g.GET, "/info/", ctrl.Info),
	}
	route.Bind(routes, controllers.ErrorCode)
}
