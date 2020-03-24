//go:generate stringer -type=QueueStatus -linecomment

package pod

import "github.com/cfhamlet/os-rq-pod/pkg/utils"

// QueueStatus type
type QueueStatus int

// QueueStatus enum
const (
	_            QueueStatus = iota
	QueueInit                // init
	QueueWorking             // working
	QueuePaused              // paused
	QueueRemoved             // removed
)

// QueueStatusMap TODO
var QueueStatusMap = map[string]QueueStatus{
	utils.Text(QueueInit):    QueueInit,
	utils.Text(QueueWorking): QueueWorking,
	utils.Text(QueuePaused):  QueuePaused,
	utils.Text(QueueRemoved): QueueRemoved,
}

// QueueStatusList TODO
var QueueStatusList = []QueueStatus{
	QueueInit,
	QueueWorking,
	QueuePaused,
	QueueRemoved,
}
