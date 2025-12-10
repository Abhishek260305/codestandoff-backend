package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOAuthConfig *oauth2.Config
	githubOAuthConfig *oauth2.Config
)

// InitGoogleOAuth initializes Google OAuth configuration
func InitGoogleOAuth() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/auth/google/callback"
	}

	googleOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// GetGoogleOAuthConfig returns the Google OAuth configuration
func GetGoogleOAuthConfig() *oauth2.Config {
	return googleOAuthConfig
}

// GenerateState generates a random state token for OAuth
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// InitGitHubOAuth initializes GitHub OAuth configuration
func InitGitHubOAuth() {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	redirectURL := os.Getenv("GITHUB_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/auth/github/callback"
	}

	githubOAuthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"user:email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		},
	}
}

// GetGitHubOAuthConfig returns the GitHub OAuth configuration
func GetGitHubOAuthConfig() *oauth2.Config {
	return githubOAuthConfig
}

// ExchangeCodeForToken exchanges OAuth code for access token
func ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	return googleOAuthConfig.Exchange(ctx, code)
}
