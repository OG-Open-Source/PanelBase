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
	"github.com/OG-Open-Source/PanelBase/internal/token_store"
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

// validateScopes checks if requested scopes are a subset of user's allowed scopes.
func validateScopes(requested models.UserPermissions, userScopes models.UserPermissions) bool {
	for resource, reqActions := range requested {
		userActions, ok := userScopes[resource]
		if !ok {
			return false // User doesn't have access to this resource at all
		}
		userActionSet := make(map[string]bool)
		for _, action := range userActions {
			userActionSet[action] = true
		}
		for _, reqAction := range reqActions {
			if !userActionSet[reqAction] {
				return false // Requested action not allowed for the user on this resource
			}
		}
	}
	return true
}

// CreateAPIToken creates metadata, generates JWT, and stores token info.
func CreateAPIToken(user models.User, payload models.CreateAPITokenPayload) (string, models.APIToken, string, error) {
	// 2. Determine and Validate Scopes
	var finalScopes models.UserPermissions
	if payload.Scopes == nil || len(payload.Scopes) == 0 {
		// If no scopes requested, inherit all user scopes
		finalScopes = user.Scopes
	} else {
		// If scopes are requested, validate they are a subset of user scopes
		if !validateScopes(payload.Scopes, user.Scopes) {
			return "", models.APIToken{}, "", errors.New("requested scopes exceed user permissions")
		}
		finalScopes = payload.Scopes // Use the validated requested scopes
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

	// 5. Create API Token Metadata using finalScopes
	apiTokenMeta := models.APIToken{
		Name:        payload.Name,
		Description: payload.Description,
		Scopes:      finalScopes, // Use the determined final scopes
		CreatedAt:   createdAtRFC3339,
		ExpiresAt:   expiresAtRFC3339,
	}

	// 6. Generate JWT using the helper function (using apiTokenMeta with finalScopes)
	signedTokenString, err := generateTokenJWT(user, tokenID, apiTokenMeta)
	if err != nil {
		return "", models.APIToken{}, "", err
	}

	// 7. Store Token Info in the token store (using apiTokenMeta with finalScopes)
	storeInfo := token_store.TokenInfo{
		UserID:    user.ID,
		Name:      apiTokenMeta.Name,
		Audience:  AudienceAPI,
		Scopes:    apiTokenMeta.Scopes, // Use scopes from metadata
		CreatedAt: apiTokenMeta.CreatedAt,
		ExpiresAt: apiTokenMeta.ExpiresAt,
	}
	if err := token_store.StoreToken(tokenID, storeInfo); err != nil {
		return "", models.APIToken{}, "", fmt.Errorf("failed to store token metadata: %w", err)
	}

	// 8. Return tokenID, metadata, and signed JWT
	return tokenID, apiTokenMeta, signedTokenString, nil
}

// parseISODurationSimpleRelativeTo is the correct duration parsing logic needed by CreateAPIToken
func parseISODurationSimpleRelativeTo(durationStr string, baseTime time.Time) (time.Time, error) {
	// Simplified parser, only handles PnYnMnD format
	if durationStr == "" || !strings.HasPrefix(durationStr, "P") {
		return time.Time{}, fmt.Errorf("invalid duration format: must start with P")
	}
	durationStr = strings.TrimPrefix(durationStr, "P")

	years, months, days := 0, 0, 0
	currentNumStr := ""

	for _, r := range durationStr {
		switch r {
		case 'Y':
			val, err := strconv.Atoi(currentNumStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid year value: %w", err)
			}
			years = val
			currentNumStr = ""
		case 'M':
			val, err := strconv.Atoi(currentNumStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid month value: %w", err)
			}
			months = val
			currentNumStr = ""
		case 'D':
			val, err := strconv.Atoi(currentNumStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid day value: %w", err)
			}
			days = val
			currentNumStr = ""
		case 'T': // Time component - ignore for this simple version
			return baseTime.AddDate(years, months, days), nil // Return date part only
		default:
			if r >= '0' && r <= '9' {
				currentNumStr += string(r)
			} else {
				return time.Time{}, fmt.Errorf("invalid character in duration: %c", r)
			}
		}
	}
	if currentNumStr != "" { // Handle number without designator at the end - Error?
		return time.Time{}, fmt.Errorf("duration ended with unassigned numbers: %s", currentNumStr)
	}

	return baseTime.AddDate(years, months, days), nil
}
