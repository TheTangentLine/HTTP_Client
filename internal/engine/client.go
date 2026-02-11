package engine

import (
	"net"
	"net/http"
	"time"
)

// newHTTPClient returns an *http.Client tuned for benchmarking:
// - keep-alives enabled
// - larger MaxIdleConns and MaxIdleConnsPerHost
func newHTTPClient(maxConns int) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          maxConns,
		MaxIdleConnsPerHost:   maxConns,
		ForceAttemptHTTP2:     true,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &http.Client{
		Timeout:   0, // we control timeouts via context / duration
		Transport: transport,
	}
}
