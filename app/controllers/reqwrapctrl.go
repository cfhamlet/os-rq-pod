package controllers

import (
	"fmt"

	"github.com/cfhamlet/os-go-netloc-rule/netloc"
	"github.com/cfhamlet/os-rq-pod/pkg/request"
	"github.com/cfhamlet/os-rq-pod/pkg/sth"
	"github.com/cfhamlet/os-rq-pod/pod/global"
	"github.com/cfhamlet/os-rq-pod/pod/reqwrap"
	"github.com/gin-gonic/gin"
)

// RequestWrapperController TODO
type RequestWrapperController struct {
	wrapper *reqwrap.RequestWrapper
}

// NewRequestWrapperController TODO
func NewRequestWrapperController(wrapper *reqwrap.RequestWrapper) *RequestWrapperController {
	return &RequestWrapperController{wrapper}
}

// AddRequestConfig TODO
func (ctrl *RequestWrapperController) AddRequestConfig(c *gin.Context) (result sth.Result, err error) {
	n := c.Query("netloc")
	reqConfig := &reqwrap.RequestConfig{}
	if err = c.ShouldBindJSON(reqConfig); err != nil {
		err = InvalidBody(fmt.Sprintf("%s", err))
		return
	}
	var nlc *netloc.Netloc
	nlc, err = reqwrap.NetlocFromString(n)
	if err == nil {
		n, r, err := ctrl.wrapper.Add(nlc, reqConfig)
		if err == nil {
			result = sth.Result{
				"replaced": sth.Result{
					"netloc": n,
					"rule":   r,
				},
				"active": sth.Result{
					"netloc": nlc,
					"rule":   reqConfig,
				},
			}
		}
	}
	return
}

// GetRequestConfig TODO
func (ctrl *RequestWrapperController) GetRequestConfig(c *gin.Context) (result sth.Result, err error) {
	n := c.Query("netloc")
	if n == "" {
		result = sth.Result{"rules": ctrl.wrapper.GetAll()}
		return
	}
	var nlc *netloc.Netloc
	nlc, err = reqwrap.NetlocFromString(n)
	if err == nil {
		n, r := ctrl.wrapper.Get(nlc)
		if n == nil {
			err = global.NotExistError(nlc.String())
		} else {
			result = sth.Result{"netloc": n, "rule": r}
		}
	} else {
		err = InvalidQuery(fmt.Sprintf("netloc: %s", err))
	}

	return
}

// MatchRequestConfig TODO
func (ctrl *RequestWrapperController) MatchRequestConfig(c *gin.Context) (result sth.Result, err error) {
	uri := c.Query("uri")
	n, r := ctrl.wrapper.MatchURI(uri)
	if n == nil {
		err = global.NotExistError(uri)
	} else {
		result = sth.Result{"netloc": n, "rule": r}
	}

	return
}

// DeleteRequestConfig TODO
func (ctrl *RequestWrapperController) DeleteRequestConfig(c *gin.Context) (result sth.Result, err error) {
	n := c.Query("netloc")
	var nlc *netloc.Netloc
	nlc, err = reqwrap.NetlocFromString(n)
	if err != nil {
		return
	}
	r, e := ctrl.wrapper.Delete(nlc)
	if e != nil {
		err = e
	} else {
		if r == nil {
			err = global.NotExistError(nlc.String())
		} else {
			result = sth.Result{"netloc": nlc, "rule": r}
		}
	}

	return
}

// TryRequestConfig TODO
func (ctrl *RequestWrapperController) TryRequestConfig(c *gin.Context) (result sth.Result, err error) {
	var raw *request.RawRequest = &request.RawRequest{}
	if err = c.ShouldBindJSON(raw); err != nil {
		err = InvalidBody(fmt.Sprintf("%s", err))
		return
	}
	var req *request.Request
	req, err = request.NewRequest(raw)
	if err == nil {
		result = sth.Result{
			"origin": req.Clone(),
		}
		nlc, rule := ctrl.wrapper.Wrap(req)
		if nlc != nil {
			result["netloc"] = nlc
			result["rule"] = rule
		}
		result["request"] = req
	}

	return
}
