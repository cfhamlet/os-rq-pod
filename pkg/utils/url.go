package utils

import (
	"errors"
	"fmt"
	"net/url"
)

// ParsedURL TODO
type ParsedURL struct {
	Parsed *url.URL
	Host   string
	Port   string
}

// NewParsedURL TODO
func NewParsedURL(rawURL string) (parsedURL *ParsedURL, err error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return parsedURL, err
	}

	if parsed.Host == "" {
		err = errors.New("empty host")
	} else {
		host := parsed.Hostname()
		port := parsed.Port()
		if host == "" {
			err = fmt.Errorf("empty host %s", parsed.Host)
		} else {
			parsedURL = &ParsedURL{parsed, host, port}
		}
	}
	if err != nil {
		err = &url.Error{Op: "parse", URL: rawURL, Err: err}
	}
	return parsedURL, err
}

// DefaultSchemePort TODO
var DefaultSchemePort = map[string]string{
	"http":  "80",
	"https": "443",
	"ftp":   "21",
	"ssh":   "22",
}
