package projects

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256-bit key

	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}

	return "bc_" + hex.EncodeToString(bytes), nil
}
