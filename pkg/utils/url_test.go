package utils

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewParsedURL(t *testing.T) {
	type Exp map[string]string
	tests := []struct {
		URL      string
		expected Exp
		isErr    bool
	}{
		{"http://www.example.com/", Exp{"Host": "www.example.com"}, false},
		{"http://www.example.com:80/", Exp{"Host": "www.example.com", "Port": "80"}, false},
		{"http:/www.example.com:80/", nil, true},
	}

	for _, test := range tests {
		parsedURL, err := NewParsedURL(test.URL)

		if test.isErr {
			assert.NotNil(t, err)
			assert.Nil(t, parsedURL)
		} else {
			v := reflect.ValueOf(*parsedURL)
			for key, value := range test.expected {
				vv := v.FieldByName(key).String()
				assert.Equal(t, vv, value)
			}
		}
	}
}

func BenchmarkNewParsedURL(b *testing.B) {
	for _, rawURL := range []string{"http://www.example.com/", "http://www.example.com:8080/"} {
		b.Run(fmt.Sprintf("NewParsedURL %s", rawURL), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = NewParsedURL(rawURL)
			}
		})
	}
}
