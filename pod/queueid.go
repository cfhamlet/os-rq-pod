package pod

import (
	"fmt"
	"strings"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
)

// RedisKeyFromQueueID TODO
func RedisKeyFromQueueID(qid sth.QueueID) string {
	return RedisQueueKeyPrefix + qid.String()
}

// QueueIDFromString TODO
func QueueIDFromString(key string) (qid sth.QueueID, err error) {
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		err = fmt.Errorf(`invalid qid %s, not "host:port:scheme"`, key)
	} else {
		qid = sth.QueueID{Host: parts[0], Port: parts[1], Scheme: parts[2]}
	}
	return
}

// QueueIDFromRedisKey TODO
func QueueIDFromRedisKey(key string) (qid sth.QueueID, err error) {
	if !strings.HasPrefix(key, RedisQueueKeyPrefix) {
		err = fmt.Errorf(`invalid redis key %s, not starts with "%s"`, key, RedisQueueKeyPrefix)
	} else {
		qid, err = QueueIDFromString(key[len(RedisQueueKeyPrefix):])
	}

	return
}
