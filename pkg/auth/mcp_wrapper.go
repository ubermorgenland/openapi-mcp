package auth

import (
	"log"
	"net/http"
	"os"
)

// SecureHTTPClientWrapper wraps HTTP requests with authentication without global state mutation
type SecureHTTPClientWrapper struct {
	client   *http.Client
	provider SecureAuthProvider
}

// NewSecureHTTPClientWrapper creates a new secure HTTP client wrapper
func NewSecureHTTPClientWrapper(client *http.Client, provider SecureAuthProvider) *SecureHTTPClientWrapper {
	if client == nil {
		client = http.DefaultClient
	}
	
	return &SecureHTTPClientWrapper{
		client:   client,
		provider: provider,
	}
}

// Do executes an HTTP request with secure authentication
func (w *SecureHTTPClientWrapper) Do(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	clonedReq := req.Clone(req.Context())
	
	// Add authentication headers
	if headers := w.provider.GetAuthHeaders(req.Context()); headers != nil {
		if os.Getenv("DEBUG") != "" {
			log.Printf("🔧 SecureHTTPClientWrapper: Adding auth headers: %+v", headers)
		}
		for key, value := range headers {
			clonedReq.Header.Set(key, value)
			if os.Getenv("DEBUG") != "" {
				log.Printf("🔧 SecureHTTPClientWrapper: Set header '%s' = '%s'", key, value)
			}
		}
	} else {
		if os.Getenv("DEBUG") != "" {
			log.Printf("⚠️ SecureHTTPClientWrapper: No auth headers returned from provider")
		}
	}
	
	// Add authentication query parameters
	if params := w.provider.GetAuthQueryParams(req.Context()); params != nil {
		if os.Getenv("DEBUG") != "" {
			log.Printf("🔧 SecureHTTPClientWrapper: Adding auth query params: %+v", params)
		}
		q := clonedReq.URL.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		clonedReq.URL.RawQuery = q.Encode()
	} else {
		if os.Getenv("DEBUG") != "" {
			log.Printf("🔧 SecureHTTPClientWrapper: No auth query params from provider")
		}
	}
	
	return w.client.Do(clonedReq)
}

// RoundTrip implements http.RoundTripper for use as a transport
type SecureRoundTripper struct {
	base     http.RoundTripper
	provider SecureAuthProvider
}

// NewSecureRoundTripper creates a new secure round tripper
func NewSecureRoundTripper(base http.RoundTripper, provider SecureAuthProvider) *SecureRoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	
	return &SecureRoundTripper{
		base:     base,
		provider: provider,
	}
}

// RoundTrip executes a single HTTP transaction with secure authentication
func (t *SecureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	clonedReq := req.Clone(req.Context())
	
	// Add authentication headers
	if headers := t.provider.GetAuthHeaders(req.Context()); headers != nil {
		for key, value := range headers {
			clonedReq.Header.Set(key, value)
		}
	}
	
	// Add authentication query parameters  
	if params := t.provider.GetAuthQueryParams(req.Context()); params != nil {
		q := clonedReq.URL.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		clonedReq.URL.RawQuery = q.Encode()
	}
	
	return t.base.RoundTrip(clonedReq)
}