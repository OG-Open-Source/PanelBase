package authutils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateSessionID creates a unique session identifier.
func GenerateSessionID() (string, error) {
	bytesLength := 16 // 16 bytes = 32 hex chars
	b := make([]byte, bytesLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes for session ID: %w", err)
	}
	return "ses_" + hex.EncodeToString(b), nil
}

// GenerateUserID creates a unique user identifier.
func GenerateUserID() (string, error) {
	bytesLength := 16 // 16 bytes = 32 hex chars
	b := make([]byte, bytesLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes for user ID: %w", err)
	}
	return "usr_" + hex.EncodeToString(b), nil
}
