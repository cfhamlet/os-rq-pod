//go:generate stringer -type=Status -linecomment

package serv

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cfhamlet/os-rq-pod/pkg/utils"
)

// StatusError TODO
type StatusError struct {
	status Status
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("bad status %s", e.status)
}

// StatusConflictError TODO
type StatusConflictError struct {
	from Status
	to   Status
}

func (e *StatusConflictError) Error() string {
	return fmt.Sprintf("conflict from %s to %s", e.from, e.to)
}

// Status type
type Status int

// Status enum
const (
	_         Status = iota
	Init             // init
	Preparing        // preparing
	Working          // working
	Paused           // paused
	Stopping         // stopping
	Stopped          // stopped
)

// StatusMap TODO
var StatusMap = map[string]Status{
	utils.Text(Init):      Init,
	utils.Text(Preparing): Preparing,
	utils.Text(Working):   Working,
	utils.Text(Paused):    Paused,
	utils.Text(Stopping):  Stopping,
	utils.Text(Stopped):   Stopped,
}

// MarshalJSON TODO
func (s Status) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(utils.Text(s))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *Status) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}

	t, ok := StatusMap[j]
	if !ok {
		return fmt.Errorf(`invalid Status value '%s'`, j)
	}

	*s = t
	return nil
}

// WorkStatus TODO
func WorkStatus(status Status) bool {
	return status == Working || status == Paused
}

// StopStatus TODO
func StopStatus(status Status) bool {
	return status == Stopping || status == Stopped
}
