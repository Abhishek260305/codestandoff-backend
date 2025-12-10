package graph

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"codestandoff/backend/internal/middleware"
	"github.com/99designs/gqlgen/graphql"
)

// ResponseWriterExtension is a gqlgen extension that provides access to ResponseWriter
type ResponseWriterExtension struct{}

var _ interface {
	graphql.ResponseInterceptor
	graphql.HandlerExtension
} = ResponseWriterExtension{}

func (e ResponseWriterExtension) ExtensionName() string {
	return "ResponseWriterExtension"
}

func (e ResponseWriterExtension) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (e ResponseWriterExtension) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	// Get ResponseWriter and Request from context (set by our middleware)
	w := middleware.GetResponseWriter(ctx)
	req := middleware.GetRequest(ctx)
	
	// Store in context for resolvers to access BEFORE they run
	if w != nil {
		ctx = context.WithValue(ctx, "responseWriter", w)
	}
	if req != nil {
		ctx = context.WithValue(ctx, "request", req)
	}
	
	// Execute the resolver
	resp := next(ctx)
	
	// After resolver execution, check if we need to set a cookie from response data
	if resp != nil && resp.Data != nil && w != nil {
		// Parse the response to check if it's a successful signup/login mutation
		var data map[string]interface{}
		if err := json.Unmarshal(resp.Data, &data); err == nil {
			// Check for signup mutation
			if signup, ok := data["signup"].(map[string]interface{}); ok {
				if token, ok := signup["token"].(string); ok && token != "" {
					// Calculate expiry (7 days from now, matching auth.GetTokenExpiry())
					expiresAt := time.Now().Add(7 * 24 * time.Hour)
					setCookieDirectly(w, token, expiresAt)
				}
			}
			// Check for login mutation
			if login, ok := data["login"].(map[string]interface{}); ok {
				if token, ok := login["token"].(string); ok && token != "" {
					// Calculate expiry (7 days from now, matching auth.GetTokenExpiry())
					expiresAt := time.Now().Add(7 * 24 * time.Hour)
					setCookieDirectly(w, token, expiresAt)
				}
			}
		}
	}
	
	return resp
}

func setCookieDirectly(w http.ResponseWriter, token string, expiresAt time.Time) {
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Domain:   "",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

