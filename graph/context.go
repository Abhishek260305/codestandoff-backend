package graph

import (
	"context"
	"net/http"
	"time"

	"codestandoff/backend/internal/middleware"
)

// GetResponseWriter retrieves the http.ResponseWriter from context
func GetResponseWriter(ctx context.Context) http.ResponseWriter {
	// Try middleware first
	if w := middleware.GetResponseWriter(ctx); w != nil {
		return w
	}
	// Try extension context
	if w, ok := ctx.Value("responseWriter").(http.ResponseWriter); ok {
		return w
	}
	return nil
}

// WithResponseWriter adds the ResponseWriter to the context
func WithResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	return middleware.WithResponseWriter(ctx, w)
}

// WithRequest adds the Request to the context
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return middleware.WithRequest(ctx, r)
}

// GetRequest retrieves the http.Request from context
func GetRequest(ctx context.Context) *http.Request {
	// Try middleware first
	if req := middleware.GetRequest(ctx); req != nil {
		return req
	}
	// Try extension context
	if req, ok := ctx.Value("request").(*http.Request); ok {
		return req
	}
	return nil
}

// SetAuthCookie sets the authentication cookie in the HTTP response
func SetAuthCookie(ctx context.Context, token string, expiresAt time.Time) {
	w := GetResponseWriter(ctx)
	if w == nil {
		// ResponseWriter not available - extension will set it from response
		return
	}

	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		Domain:   "", // Empty domain means cookie is set for the request's domain (localhost)
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	
	http.SetCookie(w, cookie)
}

// ClearAuthCookie clears the authentication cookie
func ClearAuthCookie(ctx context.Context) {
	w := GetResponseWriter(ctx)
	if w == nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

