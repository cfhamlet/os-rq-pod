package request

import (
	"github.com/cfhamlet/os-rq-pod/pkg/json"
	"github.com/cfhamlet/os-rq-pod/pkg/utils"
)

// RawRequest TODO
type RawRequest struct {
	URL      string                 `json:"url" binding:"required"`
	Method   string                 `json:"method,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Cookies  map[string]string      `json:"cookies,omitempty"`
	Body     []byte                 `json:"body,omitempty"`
	Encoding string                 `json:"encoding,omitempty"`
}

// Request TODO
type Request struct {
	*RawRequest
	*utils.ParsedURL
}

// MarshalJSON TODO
func (req *Request) MarshalJSON() ([]byte, error) {
	return json.Marshal(req.RawRequest)
}

// UnmarshalJSON TODO
func (req *Request) UnmarshalJSON(b []byte) (err error) {
	rawReq := &RawRequest{}
	err = json.Unmarshal(b, rawReq)
	if err == nil {
		parsedURL, err := utils.NewParsedURL(rawReq.URL)
		if err == nil {
			req.RawRequest = rawReq
			req.ParsedURL = parsedURL
		}
	}
	return
}

// NewRequest TODO
func NewRequest(rawReq *RawRequest) (request *Request, err error) {
	parsedURL, err := utils.NewParsedURL(rawReq.URL)
	if err == nil {
		request = &Request{rawReq, parsedURL}
	}
	return
}
