package router

import (
	"github.com/cfhamlet/os-rq-pod/app/controller"

	"github.com/cfhamlet/os-rq-pod/core"
	"github.com/cfhamlet/os-rq-pod/pkg/ginserv"
)

// InitAPIRouter TODO
func InitAPIRouter(g ginserv.RouterGroup, pod *core.Pod) {

	routers := []struct {
		HTTPFunc ginserv.IRoutesHTTPFunc
		Path     string
		F        controller.CtrlFunc
	}{
		{g.GET, "/pod/info/", controller.Info},
		{g.POST, "/pod/pause/", controller.Pause},
		{g.POST, "/pod/resume/", controller.Resume},
		{g.GET, "/pod/queues/", controller.Queues},

		{g.GET, "/queue/info/", controller.QueueInfo},
		{g.GET, "/queue/view/", controller.ViewQueue},
		{g.POST, "/queue/pause/", controller.PauseQueue},
		{g.POST, "/queue/resume/", controller.ResumeQueue},
		{g.POST, "/queue/clear/", controller.ClearQueue},
		{g.POST, "/queue/sync/", controller.SyncQueue},
		{g.DELETE, "/queue/", controller.DeleteQueue},

		{g.POST, "/request/send/", controller.AddRequest},
		{g.GET, "/request/recv/", controller.GetRequest},

		{g.GET, "/sys/process/memory/", controller.ProcessMemory},
		{g.GET, "/sys/redis/memory/", controller.RedisMemory},
		{g.GET, "/sys/redis/info/", controller.RedisInfo},
	}

	wp := controller.NewHandlerWrapper(pod)

	for _, r := range routers {
		r.HTTPFunc(r.Path, wp.Wrap(r.F))
	}
}
