package apitoken

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

const MaxTokensPerUser = 10

// --- Helper Functions ---

func generateSecureRandomHex(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func parseISODurationSimple(durationStr string) (time.Time, error) {
	if durationStr == "" || !strings.HasPrefix(durationStr, "P") {
		return time.Time{}, fmt.Errorf("invalid duration format: must start with P")
	}
	durationStr = strings.TrimPrefix(durationStr, "P")

	var years, months, days int
	currentNumStr := ""

	for _, r := range durationStr {
		switch r {
		case 'Y':
			val, err := parseInt(&currentNumStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid year value: %w", err)
			}
			years = val
		case 'M':
			val, err := parseInt(&currentNumStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid month value: %w", err)
			}
			months = val
		case 'D':
			val, err := parseInt(&currentNumStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid day value: %w", err)
			}
			days = val
		case 'W': // Weeks not commonly used with Y/M, handle separately or ignore
			// For simplicity, we can treat W as 7D if needed, but standard allows P1Y1W etc.
			// Let's return an error for now to keep it simple.
			return time.Time{}, fmt.Errorf("week designator (W) is not supported in this simple parser")
		case 'T': // Time component not supported
			return time.Time{}, fmt.Errorf("time component (T...) is not supported")
		default:
			if r >= '0' && r <= '9' {
				currentNumStr += string(r)
			} else {
				return time.Time{}, fmt.Errorf("invalid character in duration: %c", r)
			}
		}
	}

	// If the string ended with numbers without a designator
	if currentNumStr != "" {
		return time.Time{}, fmt.Errorf("duration string ended with unassigned number: %s", currentNumStr)
	}

	// Use AddDate which handles month/year rollovers correctly
	calculatedTime := time.Now().AddDate(years, months, days)
	return calculatedTime, nil
}

func parseInt(numStr *string) (int, error) {
	if *numStr == "" {
		return 0, fmt.Errorf("missing number before designator")
	}
	// Use strconv.Atoi which is cleaner
	val, err := strconv.Atoi(*numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value: %s", *numStr)
	}
	*numStr = "" // Reset for the next number
	return val, nil
}

// --- Service Function ---

func CreateAPIToken(user models.User, payload models.CreateAPITokenPayload) (string, models.APIToken, string, error) {
	// 1. Check token limit
	if len(user.API.Tokens) >= MaxTokensPerUser {
		return "", models.APIToken{}, "", fmt.Errorf("maximum token limit (%d) reached", MaxTokensPerUser)
	}

	// 2. Validate Scopes (assuming user.Scopes exists)
	if !validateScopes(payload.Scopes, user.Scopes) {
		return "", models.APIToken{}, "", errors.New("requested scopes exceed user permissions")
	}

	// 3. Calculate expiration time
	var expiresAt time.Time
	var err error
	if payload.Duration != "" {
		expiresAt, err = parseISODurationSimple(payload.Duration)
		if err != nil {
			return "", models.APIToken{}, "", fmt.Errorf("invalid duration format: %w", err)
		}
	} else {
		return "", models.APIToken{}, "", errors.New("duration is required")
	}

	expiresAtUTC := expiresAt.UTC()
	// expiresAtPtr := &expiresAtUTC // No longer needed

	// 4. Generate unique Token ID using crypto/rand
	tokenIDBytesLength := 12 // 12 bytes = 24 hex chars
	randomPart, err := generateSecureRandomHex(tokenIDBytesLength)
	if err != nil {
		return "", models.APIToken{}, "", fmt.Errorf("failed to generate token ID: %w", err)
	}
	tokenID := "tok_" + randomPart // This ID will be the map key and JWT jti

	// 5. Create API Token Metadata (No ID field here)
	apiTokenMeta := models.APIToken{
		Name:        payload.Name,
		Description: payload.Description,
		Scopes:      payload.Scopes,
		CreatedAt:   time.Now().UTC(),
		ExpiresAt:   expiresAtUTC,
	}

	// 6. Generate JWT Claims in the specified order
	claims := jwt.MapClaims{
		"aud":    "api",                         // Audience: API
		"sub":    user.ID,                       // Subject: User ID (Corrected)
		"name":   apiTokenMeta.Name,             // Name: Token's custom name
		"jti":    tokenID,                       // JWT ID: Token's unique ID (tok_...)
		"scopes": apiTokenMeta.Scopes,           // Scopes: Granted scopes for this token
		"iss":    "PanelBase",                   // Issuer: Added
		"iat":    apiTokenMeta.CreatedAt.Unix(), // Issued At
		"exp":    expiresAtUTC.Unix(),           // Expiration Time
	}

	// 7. Create and Sign JWT (assuming user.API.JwtSecret exists)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := []byte(user.API.JwtSecret)
	if len(jwtSecret) == 0 {
		return "", models.APIToken{}, "", errors.New("user JWT secret is not configured")
	}
	signedTokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", models.APIToken{}, "", fmt.Errorf("failed to sign token: %w", err)
	}

	// 8. Return tokenID, metadata, and signed JWT
	return tokenID, apiTokenMeta, signedTokenString, nil
}

func validateScopes(requested models.UserPermissions, userScopes models.UserPermissions) bool {
	// TODO: Implement robust scope validation logic.
	// Check if every scope in requested is also present in userScopes.
	return true
}
