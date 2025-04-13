package auth

import (
	"fmt"
	"time"

	"errors"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const issuer = "PanelBase"

const (
	TokenTypeWeb = "web"
	TokenTypeAPI = "api"

	JtiPrefixSession = "ses_"
	JtiPrefixToken   = "tok_"
)

// Claims defines the structure of the JWT claims according to standards.
type Claims struct {
	Scopes map[string]interface{} `json:"scopes,omitempty"`
	Name   string                 `json:"name,omitempty"` // Added name claim
	jwtv5.RegisteredClaims
	// Note: Audience ('aud') is part of RegisteredClaims
}

// GenerateToken creates a new JWT for a given user and token type.
func GenerateToken(user *models.User, secretKey string, durationMinutes int, tokenType string, apiTokenName ...string) (string, error) {
	expirationTime := time.Now().Add(time.Duration(durationMinutes) * time.Minute)

	// Ensure user.Scopes is not nil before assigning, create empty map if it is
	scopes := user.Scopes
	if scopes == nil {
		scopes = make(map[string]interface{}) // Ensure we don't pass nil scopes
	}

	// Determine Audience, JTI, and Name based on token type
	var audience []string
	var jwtID string
	var tokenName string

	switch tokenType {
	case TokenTypeWeb:
		audience = []string{TokenTypeWeb}
		jwtID = JtiPrefixSession + uuid.NewString()
		tokenName = user.Name // Use user's name for web sessions
	case TokenTypeAPI:
		audience = []string{TokenTypeAPI}
		jwtID = JtiPrefixToken + uuid.NewString()
		if len(apiTokenName) > 0 && apiTokenName[0] != "" {
			tokenName = apiTokenName[0] // Use provided API token name
		} else {
			// Fallback or error if API token name is required but not provided?
			// For now, leave it empty if not provided.
			tokenName = ""
		}
	default:
		return "", fmt.Errorf("invalid token type specified: %s", tokenType)
	}

	claims := &Claims{
		Scopes: scopes,
		Name:   tokenName,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Audience:  audience,
			ExpiresAt: jwtv5.NewNumericDate(expirationTime),
			IssuedAt:  jwtv5.NewNumericDate(time.Now()),
			Issuer:    issuer,
			Subject:   user.ID,
			ID:        jwtID,
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates a JWT string.
// It now also checks the audience if expectedAudience is provided.
func ValidateToken(tokenString string, secretKey string, expectedAudience ...string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwtv5.ParseWithClaims(tokenString, claims, func(token *jwtv5.Token) (interface{}, error) {
		// Validate the alg is what we expect: SigningMethodHS256
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		// Check for specific validation errors
		if errors.Is(err, jwtv5.ErrTokenMalformed) {
			return nil, fmt.Errorf("malformed token")
		} else if errors.Is(err, jwtv5.ErrTokenExpired) || errors.Is(err, jwtv5.ErrTokenNotValidYet) {
			return nil, fmt.Errorf("token is expired or not valid yet")
		} else {
			return nil, fmt.Errorf("couldn't handle this token: %w", err)
		}
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token (reason unknown)")
	}

	// Validate Issuer
	issue, err := claims.GetIssuer()
	if err != nil || issue != issuer {
		return nil, fmt.Errorf("invalid issuer: %v, expected %s", issue, issuer)
	}

	// Validate Subject (should exist and be non-empty user ID)
	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return nil, fmt.Errorf("invalid subject (user ID): %v", err)
	}

	// Validate JTI (should exist and be non-empty)
	if claims.RegisteredClaims.ID == "" {
		return nil, fmt.Errorf("invalid JTI (token ID): JTI is missing or empty")
	}

	// Audience (Optional but Recommended)
	if len(expectedAudience) > 0 {
		audValid := false
		// Audience is a slice of strings in RegisteredClaims
		if claims.RegisteredClaims.Audience != nil {
			for _, actual := range claims.RegisteredClaims.Audience {
				for _, expected := range expectedAudience {
					if actual == expected {
						audValid = true
						break // Found a match for this actual aud
					}
				}
				if audValid {
					break // Found a match, no need to check other actual auds
				}
			}
		}
		if !audValid {
			actualAud := claims.RegisteredClaims.Audience // Get the actual audience list
			return nil, fmt.Errorf("invalid audience: %v, expected one of %v", actualAud, expectedAudience)
		}
	}

	// Ensure Scopes map is initialized if it was omitted or null in the token
	if claims.Scopes == nil {
		claims.Scopes = make(map[string]interface{}) // Initialize to empty map
	}

	return claims, nil
}
