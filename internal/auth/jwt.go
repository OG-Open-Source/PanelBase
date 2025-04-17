package auth

import (
	"fmt"
	"time"

	"errors"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeWeb     = "web"
	TokenTypeAPI     = "api"
	JtiPrefixSession = "ses_" // For web login sessions
	JtiPrefixToken   = "tok_" // For persistent API tokens
	Issuer           = "PanelBase"
)

// Claims defines the structure of the JWT payload with a specific field order.
// Note: encoding/json doesn't guarantee struct field order, but it often follows it.
// We define all standard claims manually for potentially better order control.
type Claims struct {
	Audience  string                 `json:"aud"`    // Expected Audience (e.g., "web" or "api")
	IssuedAt  int64                  `json:"iat"`    // Issued At timestamp (Unix seconds)
	ExpiresAt int64                  `json:"exp"`    // Expiration timestamp (Unix seconds)
	Issuer    string                 `json:"iss"`    // Issuer
	ID        string                 `json:"jti"`    // JWT ID
	Subject   string                 `json:"sub"`    // Subject (User ID)
	Name      string                 `json:"name"`   // User's display name
	Scopes    map[string]interface{} `json:"scopes"` // Hierarchical scopes
}

// Implement jwt.Claims interface (GetExpirationTime, GetIssuedAt, GetNotBefore, GetIssuer, GetSubject, GetAudience)
// We need these methods for jwt.ParseWithClaims to work correctly with our custom struct.
func (c Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.ExpiresAt, 0)), nil
}
func (c Claims) GetIssuedAt() (*jwt.NumericDate, error) {
	return jwt.NewNumericDate(time.Unix(c.IssuedAt, 0)), nil
}
func (c Claims) GetNotBefore() (*jwt.NumericDate, error) {
	// NBF not used, return nil
	return nil, nil
}
func (c Claims) GetIssuer() (string, error) {
	return c.Issuer, nil
}
func (c Claims) GetSubject() (string, error) {
	return c.Subject, nil
}
func (c Claims) GetAudience() (jwt.ClaimStrings, error) {
	// Return as ClaimStrings for interface compatibility, even though we store as string
	return jwt.ClaimStrings{c.Audience}, nil
}

// GenerateToken creates a new JWT for a given user.
func GenerateToken(user *models.User, secret string, durationMinutes int, tokenType string) (string, error) {
	var jtiPrefix string
	switch tokenType {
	case TokenTypeWeb:
		jtiPrefix = JtiPrefixSession
	case TokenTypeAPI:
		jtiPrefix = JtiPrefixToken
	default:
		return "", fmt.Errorf("invalid token type: %s", tokenType)
	}
	jti := jtiPrefix + uuid.NewString()

	now := time.Now()
	expirationTime := now.Add(time.Duration(durationMinutes) * time.Minute)

	// Create the claims manually in the desired structure
	claims := Claims{
		Audience:  tokenType,             // Single string audience
		IssuedAt:  now.Unix(),            // Unix timestamp
		ExpiresAt: expirationTime.Unix(), // Unix timestamp
		Issuer:    Issuer,
		ID:        jti,
		Subject:   user.UserID,
		Name:      user.Name,
		Scopes:    user.Scopes, // Assume user.Scopes is non-nil or handled upstream
	}

	// Ensure scopes are initialized if nil
	if claims.Scopes == nil {
		claims.Scopes = make(map[string]interface{})
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT string.
func ValidateToken(tokenString string, secret string, expectedTokenType string) (*Claims, error) {
	claims := &Claims{} // Use our custom Claims struct

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		// Use standard jwt error checking
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, fmt.Errorf("malformed token")
		} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, fmt.Errorf("token is expired or not valid yet: %w", err) // Include original error
		} else {
			return nil, fmt.Errorf("token parsing error: %w", err) // Include original error
		}
	}

	if !token.Valid {
		// This case might be redundant if ParseWithClaims returns specific errors handled above,
		// but kept for robustness.
		return nil, fmt.Errorf("invalid token (validation failed)")
	}

	// --- Manual Claim Validations (using direct field access) ---

	// Validate Issuer
	if claims.Issuer != Issuer {
		return nil, fmt.Errorf("invalid issuer: %s, expected %s", claims.Issuer, Issuer)
	}

	// Validate Subject (should exist and be non-empty user ID)
	if claims.Subject == "" {
		return nil, fmt.Errorf("invalid subject (user ID): missing or empty")
	}

	// Validate JTI (should exist and be non-empty)
	if claims.ID == "" {
		return nil, fmt.Errorf("invalid JTI (token ID): missing or empty")
	}

	// Validate Audience (must match expectedTokenType)
	if claims.Audience != expectedTokenType {
		return nil, fmt.Errorf("invalid audience: %s, expected %s", claims.Audience, expectedTokenType)
	}

	// Ensure Scopes map is initialized if it was omitted or null in the token
	if claims.Scopes == nil {
		claims.Scopes = make(map[string]interface{}) // Initialize to empty map
	}

	return claims, nil
}
