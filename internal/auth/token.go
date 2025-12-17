package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// GenerateToken generates a secure random token
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetTokenExpiry returns the expiry time for a token (default: 7 days)
func GetTokenExpiry() time.Time {
	return time.Now().Add(7 * 24 * time.Hour)
}


