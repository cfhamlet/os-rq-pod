package pod

import (
	"github.com/cfhamlet/os-rq-pod/pkg/request"
)

// Limiter TODO
type Limiter struct {
}

// NewLimiter TODO
func NewLimiter(pod *Pod) *Limiter {
	return &Limiter{}
}

// AllowedNewRequest TODO
func (limiter *Limiter) AllowedNewRequest(request *request.Request) error {
	return nil
}
