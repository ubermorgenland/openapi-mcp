package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/models"
	"gopkg.in/yaml.v3"
)

type AuthContext struct {
	Token         string
	AuthType      string
	Endpoint      string
	SpecParamName string // OpenAPI spec-defined parameter name for API keys
	ApiHost       string // API host from OpenAPI spec servers
	HostHeaders   map[string]string // Host headers extracted from OpenAPI spec parameters
	
	// Cache for parsed header mappings to avoid re-parsing spec content multiple times per request
	headerMappingCache map[string]string
	
	// Store original HTTP request for header access during tool execution
	OriginalRequest *http.Request
}

type contextKey string

const authContextKey contextKey = "auth"

func CreateAuthContext(r *http.Request, doc *openapi3.T, spec *models.OpenAPISpec) *AuthContext {
	return CreateAuthContextWithToolArgs(r, doc, spec, nil)
}

// CreateAuthContextWithToolArgs creates authentication context with support for tool-level arguments
// Tool arguments take highest priority and can override database/header authentication
func CreateAuthContextWithToolArgs(r *http.Request, doc *openapi3.T, spec *models.OpenAPISpec, toolArgs map[string]any) *AuthContext {
	authCtx := &AuthContext{}

	// Extract endpoint from path
	endpoint := ""
	path := strings.Trim(r.URL.Path, "/")
	if path != "" {
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[0] != "" {
			endpoint = strings.ToLower(parts[0])
		}
	}
	authCtx.Endpoint = endpoint

	// Determine auth type from spec
	_, authType, _ := ExtractAuthSchemeFromSpec(doc)
	authCtx.AuthType = authType
	
	// Parse header mappings once and cache them in the auth context
	if spec != nil {
		log.Printf("DEBUG: Calling extractOriginalHeaderNamesFromSpec for endpoint %s", endpoint)
		authCtx.headerMappingCache = extractOriginalHeaderNamesFromSpec(spec)
		log.Printf("DEBUG: Got header mapping cache: %+v", authCtx.headerMappingCache)
	} else {
		log.Printf("DEBUG: spec is nil for endpoint %s, skipping header mapping cache", endpoint)
	}
	
	// Extract parameter name and host for API key authentication
	if authType == "apiKey" {
		authCtx.SpecParamName = extractAPIKeyParameterNameWithCache(doc, authCtx.headerMappingCache)
		authCtx.ApiHost = extractAPIHostFromSpec(doc)
		authCtx.HostHeaders = extractHostHeadersWithCache(doc, authCtx.headerMappingCache)
	}

	// Authentication Priority Hierarchy:
	// 1. Tool Arguments (highest priority) - explicit auth in tool calls
	// 2. HTTP Headers - request-specific authentication
	// 3. Database Tokens - spec-specific tokens  
	// 4. Environment Variables - fallback for compatibility
	// 5. Default Configuration - system defaults

	token := ""

	// Priority 1: Extract token from tool arguments if available
	if toolArgs != nil {
		token = extractTokenFromToolArgs(toolArgs, authType, doc)
	}

	// Priority 2: Extract token from HTTP request headers using spec-defined header names with original casing
	if token == "" {
		token = extractTokenFromRequestHeadersWithCache(r, authType, doc, authCtx.headerMappingCache)
	}

	// Priority 3: Database tokens as fallback
	if token == "" && spec != nil && spec.ApiKeyToken != nil && *spec.ApiKeyToken != "" {
		token = *spec.ApiKeyToken
	}

	// Priority 4: Environment variables as final fallback
	if token == "" {
		token = extractTokenFromEnvironment(authType)
	}

	authCtx.Token = token
	
	// Store original HTTP request for potential header access during tool execution
	authCtx.OriginalRequest = r

	return authCtx
}

func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

func FromContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey).(*AuthContext)
	return authCtx, ok
}

// ExtractAuthSchemeFromSpec extracts authentication scheme information from the OpenAPI spec
func ExtractAuthSchemeFromSpec(doc *openapi3.T) (string, string, string) {
	return ExtractAuthSchemeFromSpecWithContent(doc, "")
}

// ExtractAuthSchemeFromSpecWithContent extracts authentication scheme information from the OpenAPI spec
// and preserves original header casing from raw spec content if provided
func ExtractAuthSchemeFromSpecWithContent(doc *openapi3.T, rawSpecContent string) (string, string, string) {
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return "", "", ""
	}

	// Look for the first security scheme
	for schemeName, schemeRef := range doc.Components.SecuritySchemes {
		if schemeRef.Value != nil {
			switch schemeRef.Value.Type {
			case "apiKey":
				// Return the scheme name and the location/name info
				location := "header"
				if schemeRef.Value.In == "query" {
					location = "query"
				}
				
				headerName := schemeRef.Value.Name
				// If we have raw spec content, try to extract the original casing
				if rawSpecContent != "" && location == "header" {
					headerMapping := extractOriginalHeaderNamesFromSpec(&models.OpenAPISpec{
						SpecContent: rawSpecContent,
					})
					if originalName, exists := headerMapping[strings.ToLower(headerName)]; exists {
						headerName = originalName
					}
				}
				
				return schemeName, "apiKey", location + ":" + headerName
			case "http":
				switch schemeRef.Value.Scheme {
				case "bearer":
					return schemeName, "bearer", "header:Authorization"
				case "basic":
					return schemeName, "basic", "header:Authorization"
				}
			}
		}
	}
	return "", "", ""
}

// extractTokenFromToolArgs extracts authentication token from tool call arguments
// This allows tool-level authentication to override database/header authentication
func extractTokenFromToolArgs(toolArgs map[string]any, authType string, doc *openapi3.T) string {
	if toolArgs == nil {
		return ""
	}

	switch authType {
	case "apiKey":
		// Get the API key parameter name from the OpenAPI spec
		paramName := extractAPIKeyParameterNameFromSpec(doc)
		if paramName != "" {
			if val, ok := toolArgs[paramName]; ok {
				if strVal, ok := val.(string); ok {
					return strVal
				}
			}
		}
		
		// Fallback to common API key parameter names
		commonNames := []string{"key", "apikey", "api_key", "api-key"}
		for _, name := range commonNames {
			if val, ok := toolArgs[name]; ok {
				if strVal, ok := val.(string); ok {
					return strVal
				}
			}
		}
	case "bearer":
		// Look for bearer token in tool arguments
		// First check for Authorization field with "Bearer " prefix
		if val, ok := toolArgs["Authorization"]; ok {
			if strVal, ok := val.(string); ok {
				log.Printf("DEBUG: Found Authorization in tool args: %s", strVal[:min(50, len(strVal))]+"...")
				if strings.HasPrefix(strVal, "Bearer ") {
					token := strings.TrimPrefix(strVal, "Bearer ")
					log.Printf("DEBUG: Extracted Bearer token from tool args: %s", token[:min(20, len(token))]+"...")
					return token
				}
			}
		}
		// Then check for direct token fields
		if val, ok := toolArgs["token"]; ok {
			if strVal, ok := val.(string); ok {
				return strVal
			}
		}
		if val, ok := toolArgs["bearer_token"]; ok {
			if strVal, ok := val.(string); ok {
				return strVal
			}
		}
	}
	return ""
}

// extractAPIKeyParameterNameFromSpec extracts the API key parameter name from OpenAPI spec
func extractAPIKeyParameterNameFromSpec(doc *openapi3.T) string {
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return ""
	}

	for _, schemeRef := range doc.Components.SecuritySchemes {
		if schemeRef.Value != nil && schemeRef.Value.Type == "apiKey" {
			return schemeRef.Value.Name
		}
	}
	return ""
}

// extractAPIKeyParameterNameWithOriginalCasing extracts API key parameter name with original casing preserved
func extractAPIKeyParameterNameWithOriginalCasing(doc *openapi3.T, spec *models.OpenAPISpec) string {
	// First get the parameter name from the OpenAPI library (may be lowercase)
	normalizedParamName := extractAPIKeyParameterNameFromSpec(doc)
	if normalizedParamName == "" {
		return ""
	}
	
	// Get original casing from raw spec content
	if spec != nil {
		headerMapping := extractOriginalHeaderNamesFromSpec(spec)
		if originalName, exists := headerMapping[strings.ToLower(normalizedParamName)]; exists {
			return originalName
		}
	}
	
	// Fallback to the normalized name if we can't find the original
	return normalizedParamName
}

// extractAPIKeyParameterNameWithCache extracts API key parameter name using cached header mappings
func extractAPIKeyParameterNameWithCache(doc *openapi3.T, headerMappingCache map[string]string) string {
	// First get the parameter name from the OpenAPI library (may be lowercase)
	normalizedParamName := extractAPIKeyParameterNameFromSpec(doc)
	if normalizedParamName == "" {
		return ""
	}
	
	// Use cached header mappings if available
	if headerMappingCache != nil {
		if originalName, exists := headerMappingCache[strings.ToLower(normalizedParamName)]; exists {
			return originalName
		}
	}
	
	// Fallback to the normalized name if we can't find the original
	return normalizedParamName
}

// extractTokenFromRequestHeaders extracts authentication token from HTTP request headers
// using the exact header names defined in the OpenAPI spec's securitySchemes
func extractTokenFromRequestHeaders(r *http.Request, authType string, doc *openapi3.T) string {
	return extractTokenFromRequestHeadersWithSpec(r, authType, doc, nil)
}

// extractTokenFromRequestHeadersWithSpec extracts authentication token with spec context for original casing
func extractTokenFromRequestHeadersWithSpec(r *http.Request, authType string, doc *openapi3.T, spec *models.OpenAPISpec) string {
	switch authType {
	case "bearer":
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				return strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
	case "basic":
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Basic ") {
				return strings.TrimPrefix(authHeader, "Basic ")
			}
		}
	case "apiKey":
		// Extract the exact header name from the OpenAPI spec with original casing
		specHeaderName := extractAPIKeyHeaderFromSpecWithOriginalCasing(doc, spec)
		if specHeaderName != "" {
			if value := r.Header.Get(specHeaderName); value != "" {
				return value
			}
		}
		
		// Fallback to common header names if spec doesn't specify or header not found
		fallbackHeaders := []string{
			"Authorization",     // Generic auth header (check for non-Bearer/Basic)
			"X-API-Key",        // Common API key header
			"Api-Key",          // Alternative API key header
			"X-RapidAPI-Key",   // RapidAPI specific with correct casing
		}
		for _, header := range fallbackHeaders {
			if value := r.Header.Get(header); value != "" {
				// Handle Authorization header with API key (no Bearer/Basic prefix)
				if header == "Authorization" && !strings.HasPrefix(value, "Bearer ") && !strings.HasPrefix(value, "Basic ") {
					return value
				}
				// Handle direct API key headers
				if header != "Authorization" {
					return value
				}
			}
		}
	}
	return ""
}

// extractTokenFromRequestHeadersWithCache extracts authentication token using cached header mappings
func extractTokenFromRequestHeadersWithCache(r *http.Request, authType string, doc *openapi3.T, headerMappingCache map[string]string) string {
	switch authType {
	case "bearer":
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				return strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
	case "basic":
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if strings.HasPrefix(authHeader, "Basic ") {
				return strings.TrimPrefix(authHeader, "Basic ")
			}
		}
	case "apiKey":
		// Extract the exact header name using cached mappings
		specHeaderName := extractAPIKeyHeaderFromCache(doc, headerMappingCache)
		if specHeaderName != "" {
			if value := r.Header.Get(specHeaderName); value != "" {
				return value
			}
		}
		
		// Fallback to common header names if spec doesn't specify or header not found
		fallbackHeaders := []string{
			"Authorization",     // Generic auth header (check for non-Bearer/Basic)
			"X-API-Key",        // Common API key header
			"Api-Key",          // Alternative API key header
			"X-RapidAPI-Key",   // RapidAPI specific with correct casing
		}
		for _, header := range fallbackHeaders {
			if value := r.Header.Get(header); value != "" {
				// Handle Authorization header with API key (no Bearer/Basic prefix)
				if header == "Authorization" && !strings.HasPrefix(value, "Bearer ") && !strings.HasPrefix(value, "Basic ") {
					return value
				}
				// Handle direct API key headers
				if header != "Authorization" {
					return value
				}
			}
		}
	}
	return ""
}

// extractAPIKeyHeaderFromSpec extracts the API key header name from OpenAPI spec's securitySchemes
// with support for preserving original header casing from raw spec content
func extractAPIKeyHeaderFromSpec(doc *openapi3.T) string {
	if doc == nil || doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return ""
	}

	// Look for API key security schemes
	for _, schemeRef := range doc.Components.SecuritySchemes {
		if schemeRef.Value != nil && schemeRef.Value.Type == "apiKey" && schemeRef.Value.In == "header" {
			return schemeRef.Value.Name
		}
	}
	return ""
}

// extractAPIKeyHeaderFromSpecWithOriginalCasing extracts API key header name with original casing preserved
func extractAPIKeyHeaderFromSpecWithOriginalCasing(doc *openapi3.T, spec *models.OpenAPISpec) string {
	// First get the header name from the OpenAPI library (may be lowercase)
	normalizedHeaderName := extractAPIKeyHeaderFromSpec(doc)
	if normalizedHeaderName == "" {
		return ""
	}
	
	// Get original casing from raw spec content
	if spec != nil {
		headerMapping := extractOriginalHeaderNamesFromSpec(spec)
		if originalName, exists := headerMapping[strings.ToLower(normalizedHeaderName)]; exists {
			return originalName
		}
	}
	
	// Fallback to the normalized name if we can't find the original
	return normalizedHeaderName
}

// extractAPIKeyHeaderFromCache extracts API key header name using cached header mappings
func extractAPIKeyHeaderFromCache(doc *openapi3.T, headerMappingCache map[string]string) string {
	// First get the header name from the OpenAPI library (may be lowercase)
	normalizedHeaderName := extractAPIKeyHeaderFromSpec(doc)
	if normalizedHeaderName == "" {
		return ""
	}
	
	// Use cached header mappings if available
	if headerMappingCache != nil {
		if originalName, exists := headerMappingCache[strings.ToLower(normalizedHeaderName)]; exists {
			return originalName
		}
	}
	
	// Fallback to the normalized name if we can't find the original
	return normalizedHeaderName
}

// extractTokenFromEnvironment extracts authentication token from environment variables
// as a fallback when no request headers are provided
func extractTokenFromEnvironment(authType string) string {
	switch authType {
	case "bearer":
		if token := os.Getenv("BEARER_TOKEN"); token != "" {
			return token
		}
		// Also check generic API_KEY as fallback
		if token := os.Getenv("API_KEY"); token != "" {
			return token
		}
	case "basic":
		if token := os.Getenv("BASIC_AUTH"); token != "" {
			return token
		}
	case "apiKey":
		// Try environment variables in priority order
		envVars := []string{
			"API_KEY",           // Generic API key
			"RAPIDAPI_KEY",      // RapidAPI specific
			"X_API_KEY",         // X-API-Key variant
		}
		for _, envVar := range envVars {
			if token := os.Getenv(envVar); token != "" {
				return token
			}
		}
	}
	return ""
}
// extractAPIHostFromSpec extracts the API host from OpenAPI spec's servers section
// This is used for APIs like RapidAPI that require a specific host header (x-rapidapi-host)
func extractAPIHostFromSpec(doc *openapi3.T) string {
	if doc == nil || doc.Servers == nil || len(doc.Servers) == 0 {
		return ""
	}

	// Get the first server URL and extract the host
	server := doc.Servers[0]
	if server.URL == "" {
		return ""
	}

	// Parse the URL to extract the host
	if strings.HasPrefix(server.URL, "https://") {
		host := strings.TrimPrefix(server.URL, "https://")
		// Remove path if present
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		return host
	} else if strings.HasPrefix(server.URL, "http://") {
		host := strings.TrimPrefix(server.URL, "http://")
		// Remove path if present
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		return host
	}

	return ""
}

// extractHostHeadersFromSpec extracts required host headers from OpenAPI spec parameters
// This reads the actual header requirements from the spec and preserves original casing
func extractHostHeadersFromSpec(doc *openapi3.T) map[string]string {
	return extractHostHeadersFromSpecWithOriginalCasing(doc, nil)
}

// extractHostHeadersFromSpecWithOriginalCasing extracts host headers with original casing preserved
func extractHostHeadersFromSpecWithOriginalCasing(doc *openapi3.T, spec *models.OpenAPISpec) map[string]string {
	hostHeaders := make(map[string]string)
	
	if doc == nil || doc.Components == nil || doc.Components.Parameters == nil {
		return hostHeaders
	}
	
	// Get original header name mappings
	var headerMapping map[string]string
	if spec != nil {
		headerMapping = extractOriginalHeaderNamesFromSpec(spec)
	}
	
	// Look through all parameters for host-related headers
	for _, paramRef := range doc.Components.Parameters {
		if paramRef.Value != nil && paramRef.Value.In == "header" {
			param := paramRef.Value
			
			// Check if this is a host-related parameter
			headerName := param.Name
			if strings.Contains(strings.ToLower(headerName), "host") {
				// Use original casing if available
				originalHeaderName := headerName
				if headerMapping != nil {
					if original, exists := headerMapping[strings.ToLower(headerName)]; exists {
						originalHeaderName = original
					}
				}
				
				// Get default value from schema if available
				if param.Schema != nil && param.Schema.Value != nil && param.Schema.Value.Default != nil {
					if defaultVal, ok := param.Schema.Value.Default.(string); ok {
						hostHeaders[originalHeaderName] = defaultVal
					}
				}
				
				// If no default value, use the API host from servers
				if hostHeaders[originalHeaderName] == "" {
					hostHeaders[originalHeaderName] = extractAPIHostFromSpec(doc)
				}
			}
		}
	}
	
	return hostHeaders
}

// extractHostHeadersWithCache extracts host headers using cached header mappings
func extractHostHeadersWithCache(doc *openapi3.T, headerMappingCache map[string]string) map[string]string {
	hostHeaders := make(map[string]string)
	
	if doc == nil || doc.Components == nil || doc.Components.Parameters == nil {
		return hostHeaders
	}
	
	// Look through all parameters for host-related headers
	for _, paramRef := range doc.Components.Parameters {
		if paramRef.Value != nil && paramRef.Value.In == "header" {
			param := paramRef.Value
			
			// Check if this is a host-related parameter
			headerName := param.Name
			if strings.Contains(strings.ToLower(headerName), "host") {
				// Use cached header mappings if available
				originalHeaderName := headerName
				if headerMappingCache != nil {
					if original, exists := headerMappingCache[strings.ToLower(headerName)]; exists {
						originalHeaderName = original
					}
				}
				
				// Get default value from schema if available
				if param.Schema != nil && param.Schema.Value != nil && param.Schema.Value.Default != nil {
					if defaultVal, ok := param.Schema.Value.Default.(string); ok {
						hostHeaders[originalHeaderName] = defaultVal
					}
				}
				
				// If no default value, use the API host from servers
				if hostHeaders[originalHeaderName] == "" {
					hostHeaders[originalHeaderName] = extractAPIHostFromSpec(doc)
				}
			}
		}
	}
	
	return hostHeaders
}

// extractOriginalHeaderNamesFromSpec extracts original header names with correct casing from raw spec content
func extractOriginalHeaderNamesFromSpec(spec *models.OpenAPISpec) map[string]string {
	headerMapping := make(map[string]string)
	
	if spec == nil || spec.SpecContent == "" {
		log.Printf("DEBUG: extractOriginalHeaderNamesFromSpec - spec is nil or empty")
		return headerMapping
	}
	
	log.Printf("DEBUG: extractOriginalHeaderNamesFromSpec - parsing spec content (first 100 chars): %s", spec.SpecContent[:min(100, len(spec.SpecContent))])
	
	// Parse the raw spec content as JSON to preserve original casing
	var specData map[string]interface{}
	if err := json.Unmarshal([]byte(spec.SpecContent), &specData); err != nil {
		log.Printf("DEBUG: JSON parsing failed: %v, trying YAML", err)
		// If JSON parsing fails, try YAML parsing since database specs are stored as YAML
		if err := yaml.Unmarshal([]byte(spec.SpecContent), &specData); err != nil {
			log.Printf("DEBUG: YAML parsing also failed: %v", err)
			// If both JSON and YAML parsing fail, return empty mapping
			return headerMapping
		}
		log.Printf("DEBUG: YAML parsing succeeded")
	} else {
		log.Printf("DEBUG: JSON parsing succeeded")
	}
	
	// Check which security schemes are actually used (global security or operation security)
	usedSecuritySchemes := make(map[string]bool)
	
	// Check global security definition
	if globalSecurity, ok := specData["security"].([]interface{}); ok {
		for _, securityItem := range globalSecurity {
			if securityObj, ok := securityItem.(map[string]interface{}); ok {
				for schemeName := range securityObj {
					usedSecuritySchemes[schemeName] = true
				}
			}
		}
	}
	
	log.Printf("DEBUG: Found used security schemes: %+v", usedSecuritySchemes)
	
	// Navigate to components.securitySchemes
	components, ok := specData["components"].(map[string]interface{})
	if !ok {
		return headerMapping
	}
	
	securitySchemes, ok := components["securitySchemes"].(map[string]interface{})
	if !ok {
		return headerMapping
	}
	
	// Extract header names from security schemes that are actually used
	for schemeName, schemeData := range securitySchemes {
		// Only process security schemes that are actually referenced
		if !usedSecuritySchemes[schemeName] {
			continue
		}
		
		scheme, ok := schemeData.(map[string]interface{})
		if !ok {
			continue
		}
		
		schemeType, ok := scheme["type"].(string)
		if !ok {
			continue
		}
		
		log.Printf("DEBUG: Processing security scheme %s of type %s", schemeName, schemeType)
		
		switch schemeType {
		case "apiKey":
			// Handle API key security schemes
			in, ok := scheme["in"].(string)
			if !ok || in != "header" {
				continue
			}
			
			name, ok := scheme["name"].(string)
			if ok {
				// Map lowercase version to original casing
				headerMapping[strings.ToLower(name)] = name
				log.Printf("DEBUG: Added API key header mapping: %s -> %s", strings.ToLower(name), name)
			}
			
		case "http":
			// Handle HTTP security schemes (Bearer, Basic, etc.)
			httpScheme, ok := scheme["scheme"].(string)
			if ok && (httpScheme == "bearer" || httpScheme == "basic") {
				// Both Bearer and Basic auth use the Authorization header
				headerMapping["authorization"] = "Authorization"
				log.Printf("DEBUG: Added HTTP %s header mapping: authorization -> Authorization", httpScheme)
			}
		}
	}
	
	// Also check components.parameters for header parameters
	parameters, ok := components["parameters"].(map[string]interface{})
	if ok {
		for _, paramData := range parameters {
			param, ok := paramData.(map[string]interface{})
			if !ok {
				continue
			}
			
			in, ok := param["in"].(string)
			if !ok || in != "header" {
				continue
			}
			
			name, ok := param["name"].(string)
			if ok {
				// Map lowercase version to original casing
				headerMapping[strings.ToLower(name)] = name
			}
		}
	}
	
	log.Printf("DEBUG: extractOriginalHeaderNamesFromSpec - final header mapping: %+v", headerMapping)
	return headerMapping
}
