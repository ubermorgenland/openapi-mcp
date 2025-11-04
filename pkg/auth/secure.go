package auth

import (
	"context"
	"log"
	"net/http"
	"os"
)

// SecureAuthProvider provides authentication without global state mutation
type SecureAuthProvider interface {
	// GetAuthHeaders returns authentication headers for the given context
	GetAuthHeaders(ctx context.Context) map[string]string
	
	// GetAuthQueryParams returns authentication query parameters for the given context
	GetAuthQueryParams(ctx context.Context) map[string]string
}

// contextAuthProvider implements SecureAuthProvider using context-based authentication
type contextAuthProvider struct{}

// NewSecureAuthProvider creates a new secure authentication provider
func NewSecureAuthProvider() SecureAuthProvider {
	return &contextAuthProvider{}
}

// GetAuthHeaders extracts authentication headers from context
func (p *contextAuthProvider) GetAuthHeaders(ctx context.Context) map[string]string {
	authCtx, ok := FromContext(ctx)
	if !ok {
		if os.Getenv("DEBUG") != "" {
			log.Printf("🔍 GetAuthHeaders: No auth context found in request context")
		}
		return nil
	}
	
	if authCtx.Token == "" {
		if os.Getenv("DEBUG") != "" {
			log.Printf("🔍 GetAuthHeaders: Auth context found but token is empty: %+v", authCtx)
		}
		return nil
	}

	if os.Getenv("DEBUG") != "" {
		log.Printf("🔍 GetAuthHeaders: Auth context found with token - AuthType: %s, SpecParamName: %s", authCtx.AuthType, authCtx.SpecParamName)
	}

	headers := make(map[string]string)
	
	switch authCtx.AuthType {
	case "bearer":
		headers["Authorization"] = "Bearer " + authCtx.Token
	case "basic":
		headers["Authorization"] = "Basic " + authCtx.Token
	case "apiKey":
		// Use spec-defined parameter name if available, otherwise use common headers
		if authCtx.SpecParamName != "" {
			headers[authCtx.SpecParamName] = authCtx.Token
		} else {
			// Default to common API key headers with proper casing
			headers["Authorization"] = authCtx.Token
			headers["X-API-Key"] = authCtx.Token
			headers["Api-Key"] = authCtx.Token
			headers["X-RapidAPI-Key"] = authCtx.Token  // Use proper casing for RapidAPI
		}
		
		// Automatically add host headers as defined in the OpenAPI spec
		if authCtx.HostHeaders != nil {
			for headerName, headerValue := range authCtx.HostHeaders {
				headers[headerName] = headerValue
			}
		}
	}
	
	return headers
}

// GetAuthQueryParams extracts authentication query parameters from context
func (p *contextAuthProvider) GetAuthQueryParams(ctx context.Context) map[string]string {
	authCtx, ok := FromContext(ctx)
	if !ok || authCtx.Token == "" || authCtx.AuthType != "apiKey" {
		return nil
	}

	params := make(map[string]string)
	
	// Prioritize spec-defined parameter name for accuracy
	if authCtx.SpecParamName != "" {
		params[authCtx.SpecParamName] = authCtx.Token
	} else {
		// Dynamic fallback based on common API patterns
		// Order matters: most specific to most generic
		fallbackNames := []string{"key", "api_key", "apikey"}
		for _, name := range fallbackNames {
			params[name] = authCtx.Token
		}
	}
	
	return params
}

// SecureRequestModifier modifies HTTP requests with authentication without using environment variables
type SecureRequestModifier struct {
	provider SecureAuthProvider
}

// NewSecureRequestModifier creates a new secure request modifier
func NewSecureRequestModifier(provider SecureAuthProvider) *SecureRequestModifier {
	return &SecureRequestModifier{
		provider: provider,
	}
}

// ModifyRequest adds authentication to an HTTP request using context
func (m *SecureRequestModifier) ModifyRequest(req *http.Request) {
	ctx := req.Context()
	
	// Add authentication headers
	if headers := m.provider.GetAuthHeaders(ctx); headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
	
	// Add authentication query parameters
	if params := m.provider.GetAuthQueryParams(ctx); params != nil {
		q := req.URL.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}
}