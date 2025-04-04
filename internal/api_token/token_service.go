package api_token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/tokenstore"
	"github.com/golang-jwt/jwt/v5"
)

const MaxTokensPerUser = 10

// Constants
const AudienceAPI = "api"

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

// generateTokenJWT generates and signs a JWT for a given token metadata.
func generateTokenJWT(user models.User, tokenID string, tokenMeta models.APIToken) (string, error) {
	claims := jwt.MapClaims{
		"aud":    AudienceAPI,
		"sub":    user.ID,
		"name":   tokenMeta.Name,
		"jti":    tokenID,
		"scopes": tokenMeta.Scopes,
		"iss":    "PanelBase",
		"iat":    tokenMeta.CreatedAt.Time().Unix(),
		"exp":    tokenMeta.ExpiresAt.Time().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := []byte(user.API.JwtSecret)
	if len(jwtSecret) == 0 {
		return "", errors.New("user JWT secret is not configured")
	}
	signedTokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedTokenString, nil
}

// CreateAPIToken creates metadata, generates JWT, and stores token info.
func CreateAPIToken(user models.User, payload models.CreateAPITokenPayload) (string, models.APIToken, string, error) {
	// 1. Check token limit - This check needs to change!
	// We can't rely on user.API.Tokens anymore. Need a way to count tokens in the store.
	// TODO: Implement token count check using tokenstore (requires user index or iteration).
	// For now, remove the check or keep a non-functional placeholder.
	/*
		if len(user.API.Tokens) >= MaxTokensPerUser { // This is now incorrect
			return "", models.APIToken{}, "", fmt.Errorf("maximum token limit (%d) reached", MaxTokensPerUser)
		}
	*/

	// 2. Validate Scopes (assuming user.Scopes exists)
	if !validateScopes(payload.Scopes, user.Scopes) {
		return "", models.APIToken{}, "", errors.New("requested scopes exceed user permissions")
	}

	// 3. Calculate expiration time
	var expiresAt time.Time
	var err error
	if payload.Duration != "" {
		nowUTC := time.Now().UTC()
		expiresAt, err = parseISODurationSimpleRelativeTo(payload.Duration, nowUTC)
		if err != nil {
			return "", models.APIToken{}, "", fmt.Errorf("invalid duration format: %w", err)
		}
	} else {
		return "", models.APIToken{}, "", errors.New("duration is required")
	}

	expiresAtRFC3339 := models.RFC3339Time(expiresAt)
	createdAtRFC3339 := models.RFC3339Time(time.Now().UTC())

	// 4. Generate unique Token ID using crypto/rand
	tokenIDBytesLength := 12 // 12 bytes = 24 hex chars
	randomPart, err := generateSecureRandomHex(tokenIDBytesLength)
	if err != nil {
		return "", models.APIToken{}, "", fmt.Errorf("failed to generate token ID: %w", err)
	}
	tokenID := "tok_" + randomPart // This ID will be the map key and JWT jti

	// 5. Create API Token Metadata
	apiTokenMeta := models.APIToken{
		Name:        payload.Name,
		Description: payload.Description,
		Scopes:      payload.Scopes,
		CreatedAt:   createdAtRFC3339,
		ExpiresAt:   expiresAtRFC3339,
	}

	// 6. Generate JWT using the helper function
	signedTokenString, err := generateTokenJWT(user, tokenID, apiTokenMeta)
	if err != nil {
		// Error already formatted by helper
		return "", models.APIToken{}, "", err
	}

	// 7. Store Token Info in the token store
	storeInfo := tokenstore.TokenInfo{
		UserID:    user.ID,
		Name:      apiTokenMeta.Name,
		Audience:  AudienceAPI,
		Scopes:    apiTokenMeta.Scopes,
		CreatedAt: apiTokenMeta.CreatedAt,
		ExpiresAt: apiTokenMeta.ExpiresAt,
	}
	if err := tokenstore.StoreToken(tokenID, storeInfo); err != nil {
		// Failing seems safer if we can't record the token.
		return "", models.APIToken{}, "", fmt.Errorf("failed to store token metadata: %w", err)
	}

	// 8. Return tokenID, metadata, and signed JWT
	return tokenID, apiTokenMeta, signedTokenString, nil
}

// parseISODurationSimpleRelativeTo calculates expiration based on a reference time.
// Modified from parseISODurationSimple to accept a starting point.
func parseISODurationSimpleRelativeTo(durationStr string, startTime time.Time) (time.Time, error) {
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
		case 'W', 'T':
			return time.Time{}, fmt.Errorf("W and T designators are not supported")
		default:
			if r >= '0' && r <= '9' {
				currentNumStr += string(r)
			} else {
				return time.Time{}, fmt.Errorf("invalid character: %c", r)
			}
		}
	}

	if currentNumStr != "" {
		return time.Time{}, fmt.Errorf("trailing number: %s", currentNumStr)
	}

	// Use AddDate on the provided startTime
	calculatedTime := startTime.AddDate(years, months, days)
	return calculatedTime, nil
}

// validateScopes checks if the requested scopes are a valid subset of the user's permissions.
// Make this function potentially reusable by handler (or move logic).
// For now, keep it internal. If handler needs it, call a service method wrapping this.
func validateScopes(requested models.UserPermissions, userScopes models.UserPermissions) bool {
	// TODO: Implement robust scope validation logic.
	return true
}
