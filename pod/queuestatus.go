//go:generate stringer -type=QueueStatus -linecomment

package pod

import (
	"bytes"
	"fmt"

	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
)

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

// MarshalJSON TODO
func (s QueueStatus) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(utils.Text(s))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON TODO
func (s *QueueStatus) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}

	t, ok := QueueStatusMap[j]
	if !ok {
		return fmt.Errorf(`invalid QueueStatus value '%s'`, j)
	}

	*s = t
	return nil
}
