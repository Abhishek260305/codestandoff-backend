package middleware

import (
	"context"
	"net/http"
)

type responseWriterKey struct{}
type requestKey struct{}

// GetResponseWriter retrieves the http.ResponseWriter from context
func GetResponseWriter(ctx context.Context) http.ResponseWriter {
	if rw, ok := ctx.Value(responseWriterKey{}).(http.ResponseWriter); ok {
		return rw
	}
	return nil
}

// GetRequest retrieves the http.Request from context
func GetRequest(ctx context.Context) *http.Request {
	if req, ok := ctx.Value(requestKey{}).(*http.Request); ok {
		return req
	}
	return nil
}

// WithResponseWriter adds the ResponseWriter to the context
func WithResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, responseWriterKey{}, w)
}

// WithRequest adds the Request to the context
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, requestKey{}, r)
}

