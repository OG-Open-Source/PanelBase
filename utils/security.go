package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateSecurityEntry() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}