package auth

import (
	"fmt"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// Claims defines the structure of the JWT claims.
type Claims struct {
	UserID   string              `json:"user_id"` // Store as string for simplicity in claims
	Username string              `json:"username"`
	Scopes   map[string][]string `json:"scopes"`
	jwt.RegisteredClaims
}

// GenerateToken creates a new JWT for a given user.
func GenerateToken(user *models.User, secretKey string, durationMinutes int) (string, error) {
	expirationTime := time.Now().Add(time.Duration(durationMinutes) * time.Minute)
	claims := &Claims{
		UserID:   user.ID.String(),
		Username: user.Username,
		Scopes:   user.Scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates a JWT string.
// It returns the claims if the token is valid, otherwise an error.
func ValidateToken(tokenString string, secretKey string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is what we expect: SigningMethodHS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
 