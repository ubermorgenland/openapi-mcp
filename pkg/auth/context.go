package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/models"
)

type AuthContext struct {
	Token    string
	AuthType string
	Endpoint string
}

type contextKey string

const authContextKey contextKey = "auth"

func CreateAuthContext(r *http.Request, doc *openapi3.T, spec *models.OpenAPISpec) *AuthContext {
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

	// Get token from spec or environment
	if spec != nil && spec.ApiKeyToken != nil && *spec.ApiKeyToken != "" {
		authCtx.Token = *spec.ApiKeyToken
	}

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
				return schemeName, "apiKey", location + ":" + schemeRef.Value.Name
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