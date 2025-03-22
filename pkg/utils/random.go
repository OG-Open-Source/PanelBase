package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"strconv"
)

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) (string, error) {
	// Limit length to avoid JWT key format issues
	if length > 128 {
		length = 128
	}

	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateRandomPort generates a random port within the specified range
func GenerateRandomPort(min, max int) (int, error) {
	// Generate a random integer within the specified range
	delta := max - min
	n, err := rand.Int(rand.Reader, big.NewInt(int64(delta)))
	if err != nil {
		return 0, err
	}

	return min + int(n.Int64()), nil
}

// IsPortAvailable checks if the specified port is available
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// FindAvailablePort finds an available port
func FindAvailablePort(min, max int) (int, error) {
	// Try up to 100 times
	for i := 0; i < 100; i++ {
		port, err := GenerateRandomPort(min, max)
		if err != nil {
			return 0, err
		}

		if IsPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("could not find available port in given range")
}
