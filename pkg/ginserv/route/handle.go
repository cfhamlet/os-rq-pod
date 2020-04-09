package route

import (
	"net/http"

	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/gin-gonic/gin"
)

// HandlerFunc TODO
type HandlerFunc func(*gin.Context) (sth.Result, error)

// ErrorCodeFunc TODO
type ErrorCodeFunc func(err error) int

// Bind TODO
func Bind(routes []*Route, efunc ErrorCodeFunc) {
	for _, route := range routes {
		route.Bind(efunc)
	}
}

// HandleError TODO
func HandleError(f HandlerFunc, efunc ErrorCodeFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := http.StatusOK
		result, err := f(c)

		if err != nil {
			code = http.StatusInternalServerError
			if efunc != nil {
				code = efunc(err)
			}

			if result == nil {
				result = sth.Result{}
			}

			result["err"] = err.Error()
		} else if result == nil {
			return
		}
		c.JSON(code, result)
	}
}
