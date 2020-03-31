package router

import (
	"github.com/cfhamlet/os-rq-pod/app/controller"

	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	core "github.com/cfhamlet/os-rq-pod/pod"
)

// InitAPIRouter TODO
func InitAPIRouter(g ginserv.RouterGroup, serv *core.Core) {
	ctrl := controller.New(serv)
	routers := []ginserv.RouterRecord{
		{M: g.GET, P: "/queue/info/", H: ctrl.QueueInfo},
		{M: g.GET, P: "/queue/view/", H: ctrl.ViewQueue},
		{M: g.POST, P: "/queue/pause/", H: ctrl.PauseQueue},
		{M: g.POST, P: "/queue/resume/", H: ctrl.ResumeQueue},
		{M: g.POST, P: "/queue/clear/", H: ctrl.ClearQueue},
		{M: g.POST, P: "/queue/sync/", H: ctrl.SyncQueue},
		{M: g.DELETE, P: "/queue/", H: ctrl.DeleteQueue},

		{M: g.POST, P: "/queues/", H: ctrl.Queues},
		{M: g.GET, P: "/queues/view/", H: ctrl.ViewQueues},

		{M: g.POST, P: "/request/push/", H: ctrl.PushRequest},
		{M: g.POST, P: "/request/pop/", H: ctrl.PopRequest},

		{M: g.GET, P: "/system/info/", H: ctrl.Info},
		{M: g.GET, P: "/system/info/process/memory/", H: ctrl.ProcessMemory},
		{M: g.GET, P: "/system/info/redis/", H: ctrl.RedisInfo},

		{M: g.POST, P: "/system/ctrl/pause/", H: ctrl.Pause},
		{M: g.POST, P: "/system/ctrl/resume/", H: ctrl.Resume},
	}

	ginserv.Bind(routers, controller.ErrorCode)
}
