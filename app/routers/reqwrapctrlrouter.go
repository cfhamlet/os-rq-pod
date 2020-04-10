package routers

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv/route"
)

// RouteRequestWrapperCtrl TODO
func RouteRequestWrapperCtrl(root ginserv.RouterGroup, ctrl *controllers.RequestWrapperController) {
	g := root.Group("/request/config/")
	routes := []*route.Route{
		route.New(g.GET, "/", ctrl.GetRequestConfig),
		route.New(g.POST, "/", ctrl.AddRequestConfig),
		route.New(g.DELETE, "/", ctrl.DeleteRequestConfig),
		route.New(g.POST, "/try/", ctrl.TryRequestConfig),
		route.New(g.GET, "/match/", ctrl.MatchRequestConfig),
	}
	route.Bind(routes, controllers.ErrorCode)

}
