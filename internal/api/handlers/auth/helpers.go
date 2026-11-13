package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

const refreshTokenTTL = 30 * 24 * time.Hour

func generateRefreshToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("auth: generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}
