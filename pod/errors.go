package pod

import "fmt"

// QueueNotExistError TODO
type QueueNotExistError string

func (e QueueNotExistError) Error() string {
	if e != "" {
		return "not exist " + string(e)
	}
	return "not exist"
}

// QueueNotExist TODO
const QueueNotExist = QueueNotExistError("")

// UnavailableError TODO
type UnavailableError string

func (e UnavailableError) Error() string {
	return fmt.Sprintf("unavailable %s", string(e))
}

// ExceedLimitError TODO
type ExceedLimitError string

func (e ExceedLimitError) Error() string {
	return fmt.Sprintf("exceed limit %s", string(e))
}
