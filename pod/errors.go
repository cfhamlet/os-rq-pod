package pod

import "fmt"

// QueueNotExistError TODO
type QueueNotExistError string

func (e QueueNotExistError) Error() string {
	return fmt.Sprintf("not exist %s", string(e))
}

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
