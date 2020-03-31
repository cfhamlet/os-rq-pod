package controller

import (
	"net/http"
	"net/url"

	core "github.com/cfhamlet/os-rq-pod/pod"
)

// ErrorCode TODO
func ErrorCode(err error) int {
	var code int
	switch err.(type) {
	case *url.Error, InvalidQuery, InvalidBody:
		code = http.StatusBadRequest
	case core.NotExistError:
		code = http.StatusNotFound
	case core.UnavailableError, core.ExceedLimitError:
		code = http.StatusNotAcceptable
	default:
		code = http.StatusInternalServerError
	}
	return code
}
