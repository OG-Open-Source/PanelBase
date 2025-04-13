package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	// Define the character set for generated passwords
	passwordCharset       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-="
	defaultPasswordLength = 16
)

// GenerateRandomPassword creates a random password string of a specified length.
func GenerateRandomPassword(length int) (string, error) {
	if length <= 0 {
		length = defaultPasswordLength
	}

	b := make([]byte, length)
	max := big.NewInt(int64(len(passwordCharset)))

	for i := range b {
		randIndex, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index for password: %w", err)
		}
		b[i] = passwordCharset[randIndex.Int64()]
	}

	return string(b), nil
}
