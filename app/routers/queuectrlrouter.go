package routers

import (
	"github.com/cfhamlet/os-rq-pod/app/controllers"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv/route"
)

// RouteQueueCtrl TODO
func RouteQueueCtrl(root ginserv.RouterGroup, ctrl *controllers.QueueController) {
	g := root.Group("/queue/")
	routes := []*route.Route{
		route.New(g.GET, "/info/", ctrl.QueueInfo),
		route.New(g.GET, "/view/", ctrl.ViewQueue),
		route.New(g.POST, "/pause/", ctrl.PauseQueue),
		route.New(g.POST, "/resume/", ctrl.ResumeQueue),
		route.New(g.POST, "/clear/", ctrl.ClearQueue),
		route.New(g.POST, "/sync/", ctrl.SyncQueue),
		route.New(g.DELETE, "/", ctrl.DeleteQueue),
	}
	route.Bind(routes, controllers.ErrorCode)
}
