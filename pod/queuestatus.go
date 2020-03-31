//go:generate stringer -type=QueueStatus -linecomment

package pod

import (
	"bytes"
	"fmt"

	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
)

// QueueStatus TODO
type QueueStatus int

// QueueStatus enum
const (
	_       QueueStatus = iota
	Working             // working
	Paused              // paused
)

// QueueStatusList TODO
var QueueStatusList = []QueueStatus{Working, Paused}

// QueueStatusMap TODO
var QueueStatusMap = map[string]QueueStatus{
	utils.Text(Working): Working,
	utils.Text(Paused):  Paused,
}

// MarshalJSON TODO
func (s QueueStatus) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(utils.Text(s))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *QueueStatus) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}

	t, ok := QueueStatusMap[j]
	if !ok {
		return fmt.Errorf(`invalid Status value '%s'`, j)
	}

	*s = t
	return nil
}
