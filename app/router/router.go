package router

import (
	"github.com/cfhamlet/os-rq-pod/app/controller"

	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
	core "github.com/cfhamlet/os-rq-pod/pod"
)

// InitAPIRouter TODO
func InitAPIRouter(g ginserv.RouterGroup, pod *core.Pod) {

	routers := []struct {
		HTTPFunc ginserv.IRoutesHTTPFunc
		Path     string
		F        controller.CtrlFunc
	}{
		{g.GET, "/queue/info/", controller.QueueInfo},
		{g.GET, "/queue/view/", controller.ViewQueue},
		{g.POST, "/queue/pause/", controller.PauseQueue},
		{g.POST, "/queue/resume/", controller.ResumeQueue},
		{g.POST, "/queue/clear/", controller.ClearQueue},
		{g.POST, "/queue/sync/", controller.SyncQueue},
		{g.DELETE, "/queue/", controller.DeleteQueue},

		{g.GET, "/queues/", controller.Queues},
		{g.GET, "/queues/view/", controller.ViewQueues},

		{g.POST, "/request/push/", controller.AddRequest},
		{g.GET, "/request/pop/", controller.GetRequest},

		{g.GET, "/sys/info/", controller.Info},
		{g.GET, "/sys/info/process/memory/", controller.ProcessMemory},
		{g.GET, "/sys/info/redis/memory/", controller.RedisMemory},
		{g.GET, "/sys/info/redis/", controller.RedisInfo},

		{g.POST, "/sys/ctrl/pause/", controller.Pause},
		{g.POST, "/sys/ctrl/resume/", controller.Resume},
	}

	wp := controller.NewHandlerWrapper(pod)

	for _, r := range routers {
		r.HTTPFunc(r.Path, wp.Wrap(r.F))
	}
}
