package client

import "net/http"

func NewDefaultHTTPTransport() http.RoundTripper {
	// Override transport idle connection settings
	//
	// See this conversation around why `.Clone()` is used here:
	// https://github.com/golang/go/issues/26013
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.IdleConnTimeout = DefaultIdleConnTimeout
	defaultTransport.MaxIdleConns = DefaultMaxConnections
	defaultTransport.MaxIdleConnsPerHost = DefaultMaxConnections

	return defaultTransport
}
