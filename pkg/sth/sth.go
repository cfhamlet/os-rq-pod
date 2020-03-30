package sth

import (
	"encoding/json"
	"strings"

	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
	"github.com/segmentio/fasthash/fnv1a"
)

// QueueID TODO
type QueueID struct {
	Host   string `json:"host"`
	Port   string `json:"port"`
	Scheme string `json:"scheme"`
}

func (qid QueueID) String() string {
	return strings.Join([]string{qid.Host, qid.Port, qid.Scheme}, ":")
}

// ItemID TODO
func (qid QueueID) ItemID() uint64 {
	return fnv1a.HashString64(qid.String())
}

// MarshalJSON TODO
func (qid QueueID) MarshalJSON() ([]byte, error) {
	return json.Marshal(qid.String())
}

// QueueIDFromRequest TODO
func QueueIDFromRequest(req *request.Request) QueueID {
	return CreateQueueID(req.Host, req.Port, req.Parsed.Scheme)
}

// CreateQueueID TODO
func CreateQueueID(host, port, scheme string) QueueID {
	defaultPort, ok := utils.DefaultSchemePort[scheme]
	if ok {
		if defaultPort == port {
			port = ""
		}
	}
	return QueueID{Host: host, Port: port, Scheme: scheme}
}

// ResultAndErrorFunc TODO
type ResultAndErrorFunc func() (Result, error)

// Result TODO
type Result map[string]interface{}
