package controllers

import (
	"fmt"
	"strings"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
)

// CallByQueueID TODO
type CallByQueueID func(sth.QueueID) (sth.Result, error)

// QueueIDFromQuery TODO
func QueueIDFromQuery(q string) (qid sth.QueueID, err error) {
	if strings.Contains(q, "://") {
		p, err := utils.NewParsedURL(q)
		if err == nil {
			qid = sth.CreateQueueID(p.Host, p.Port, p.Parsed.Scheme)
		}
	} else {
		s := strings.Split(q, ":")
		switch len(s) {
		case 1:
			qid = sth.CreateQueueID(q, "", "http")
		case 2:
			qid = sth.CreateQueueID(s[0], s[1], "http")
		case 3: // QueueID
			qid = sth.CreateQueueID(s[0], s[1], s[2])
		default:
			err = InvalidQuery(fmt.Sprintf("%q", q))
		}
	}

	return
}
