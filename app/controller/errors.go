package controller

import "fmt"

// InvalidQuery TODO
type InvalidQuery string

func (e InvalidQuery) Error() string {
	return fmt.Sprintf("invalid %s", string(e))
}

// InvalidBody TODO
type InvalidBody string

func (e InvalidBody) Error() string {
	return fmt.Sprintf("invalid %s", string(e))
}
