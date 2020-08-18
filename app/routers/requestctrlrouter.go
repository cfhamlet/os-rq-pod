package routers

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv/route"
)

// RouteRequestCtrl TODO
func RouteRequestCtrl(root ginserv.RouterGroup, ctrl *controllers.RequestController) {
	g := root.Group("/queue/")
	routes := []*route.Route{
		route.New(g.POST, "/enqueue/", ctrl.EnqueueRequest),
		route.New(g.POST, "/dequeue/", ctrl.DequeueRequest),
	}
	route.Bind(routes, controllers.ErrorCode)
}
