package controller

import (
	"net/http"
	"net/url"

	core "github.com/cfhamlet/os-rq-pod/pod"
	"github.com/gin-gonic/gin"
)

// CtrlFunc TODO
type CtrlFunc func(*gin.Context, *core.Pod) (core.Result, error)

// NewHandlerWrapper TODO
func NewHandlerWrapper(pod *core.Pod) *HandlerWrapper {
	return &HandlerWrapper{pod}
}

// HandlerWrapper TODO
type HandlerWrapper struct {
	pod *core.Pod
}

// Wrap TODO
func (wp *HandlerWrapper) Wrap(f CtrlFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := http.StatusOK
		result, err := f(c, wp.pod)

		if err != nil {
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

			if result == nil {
				result = core.Result{}
			}

			result["err"] = err.Error()
		} else if result == nil {
			return
		}
		c.JSON(code, result)
	}
}
