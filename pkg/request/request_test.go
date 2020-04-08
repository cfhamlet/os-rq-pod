package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClone(t *testing.T) {
	req, _ := NewRequest(&RawRequest{
		URL: "http://www.google.com/",
		Meta: map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
			"k3": "v3",
			"k4": "v4",
		},
		Headers: map[string]string{
			"k1": "v1",
			"k2": "v2",
			"k3": "v3",
			"k4": "v4",
		},
		Cookies: map[string]string{
			"k1": "v1",
			"k2": "v2",
			"k3": "v3",
			"k4": "v4",
		},
	})
	new := req.Clone()
	if req == new {
		t.Error("not new request")
	}
	assert.Equal(t, req, new)
}

func BenchmarkClone(b *testing.B) {
	req, _ := NewRequest(&RawRequest{
		URL: "http://www.google.com/",
		Meta: map[string]interface{}{
			"k1": "v1",
			"k2": "v2",
			"k3": "v3",
			"k4": "v4",
		},
		Headers: map[string]string{
			"k1": "v1",
			"k2": "v2",
			"k3": "v3",
			"k4": "v4",
		},
		Cookies: map[string]string{
			"k1": "v1",
			"k2": "v2",
			"k3": "v3",
			"k4": "v4",
		},
	})
	for i := 0; i < b.N; i++ {
		req.Clone()
	}
}
