package graph

import (
	"context"
	"net/http"
)

// getCookieFromContext retrieves the auth cookie from the request
func getCookieFromContext(ctx context.Context) (*http.Cookie, error) {
	req := GetRequest(ctx)
	if req == nil {
		return nil, nil
	}
	
	return req.Cookie("auth_token")
}

