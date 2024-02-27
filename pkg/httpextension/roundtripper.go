package httpextension

import (
	"net/http"
	"time"
)

// CircuitBreakerRoundTripper - custom round tripper to enable forwarding https req without generating TLS cert
type CircuitBreakerRoundTripper struct {
	DefaultTransport http.RoundTripper
	RequestTimeout   string
}

// RoundTrip - implementation of round tripper interface, saves origin protocol, then forces http between host and
// gofuse pod. Adds request timeout header.
func (rt *CircuitBreakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Request-Timeout", rt.RequestTimeout)
	req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)
	req.URL.Scheme = "http"
	return rt.DefaultTransport.RoundTrip(req)
}

// DefaultHTTPClient - returns default http.Client with gofuse custom round tripper extension builtin
func DefaultHTTPClient(localTimeout time.Duration, reqTimeout time.Duration) *http.Client {
	return &http.Client{
		Transport: NewCircuitBreakerRT(reqTimeout),
		Timeout:   localTimeout,
	}
}

// NewCircuitBreakerRT - returns custom round-tripper, that enables communication with gofuse
func NewCircuitBreakerRT(timeout time.Duration) *CircuitBreakerRoundTripper {
	return &CircuitBreakerRoundTripper{
		DefaultTransport: http.DefaultTransport,
		RequestTimeout:   timeout.String(),
	}
}
