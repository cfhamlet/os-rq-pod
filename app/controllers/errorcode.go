package controllers

import (
	"net/http"
	"net/url"

	"github.com/cfhamlet/os-rq-pod/pod/global"
)

// ErrorCode TODO
func ErrorCode(err error) int {
	var code int
	switch err.(type) {
	case *url.Error, InvalidQuery, InvalidBody:
		code = http.StatusBadRequest
	case global.NotExistError:
		code = http.StatusNotFound
	case global.UnavailableError, global.ExceedLimitError:
		code = http.StatusNotAcceptable
	default:
		code = http.StatusInternalServerError
	}
	return code
}
