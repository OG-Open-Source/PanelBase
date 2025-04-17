package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	idCharset            = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	defaultIDLength      = 8 // Default length for the random part
	secretCharset        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{};:,.<>/?"
	UserIDPrefix         = "usr_" // Prefix for user IDs
	UserIDRandomLength   = 12     // Random part length for user IDs
	PluginIDPrefix       = "plg_" // Prefix for plugin IDs
	PluginIDRandomLength = 12     // Random part length for plugin IDs
)

// GenerateRandomID creates a random string of a specified length using the idCharset.
func GenerateRandomID(length int) (string, error) {
	if length <= 0 {
		length = defaultIDLength
	}

	b := make([]byte, length)
	max := big.NewInt(int64(len(idCharset)))

	for i := range b {
		randIndex, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index for ID: %w", err)
		}
		b[i] = idCharset[randIndex.Int64()]
	}

	return string(b), nil
}

// GeneratePrefixedID creates a random ID with a given prefix.
// Example: GeneratePrefixedID("usr_", 8)
func GeneratePrefixedID(prefix string, randomLength int) (string, error) {
	randomPart, err := GenerateRandomID(randomLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate random part for prefixed ID: %w", err)
	}
	return prefix + randomPart, nil
}

// GenerateSecureRandomString creates a random string of a specified length using the secretCharset.
// Suitable for generating initial passwords or API secrets.
func GenerateSecureRandomString(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	b := make([]byte, length)
	max := big.NewInt(int64(len(secretCharset)))

	for i := range b {
		randIndex, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index for secret: %w", err)
		}
		b[i] = secretCharset[randIndex.Int64()]
	}

	return string(b), nil
}
