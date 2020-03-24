//go:generate stringer -type=Status -linecomment

package pod

import (
	"bytes"
	"fmt"

	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
)

// Status type
type Status int

// Status enum
const (
	_         Status = iota
	Init             // init
	Preparing        // preparing
	Working          // working
	Paused           // paused
	Stopping         //stopping
	Stopped          //stopped
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

// UnmarshalJSON TODO
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
