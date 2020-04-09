package route

import "github.com/gin-gonic/gin"

// IRouteFunc TODO
type IRouteFunc func(string, ...gin.HandlerFunc) gin.IRoutes

// Route TODO
type Route struct {
	m IRouteFunc
	p string
	h HandlerFunc
}

// New TODO
func New(metodFunc IRouteFunc, relativePath string, handlerFunc HandlerFunc) *Route {
	return &Route{metodFunc, relativePath, handlerFunc}
}

// Bind TODO
func (route *Route) Bind(efunc ErrorCodeFunc) {
	route.m(route.p, HandleError(route.h, efunc))
}
