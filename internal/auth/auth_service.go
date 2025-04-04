package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"     // For JWTClaims, Audience constants
	"github.com/OG-Open-Source/PanelBase/internal/middleware" // Import middleware
	"github.com/OG-Open-Source/PanelBase/internal/models"     // For User structure
	"github.com/OG-Open-Source/PanelBase/internal/server"     // For Response functions
	"github.com/OG-Open-Source/PanelBase/internal/tokenstore" // Import token store
	"github.com/OG-Open-Source/PanelBase/internal/user"       // Import the user service
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginPayload defines the structure for the login request body.
type LoginPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// generateSessionID creates a unique session identifier.
func generateSessionID() (string, error) {
	bytesLength := 16 // 16 bytes = 32 hex chars
	b := make([]byte, bytesLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes for session ID: %w", err)
	}
	return "ses_" + hex.EncodeToString(b), nil
}

// LoginHandler handles user login requests.
func LoginHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload LoginPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// 1. Load user data using the user service
		userInstance, userExists, err := user.GetUserByUsername(payload.Username)
		if err != nil {
			// Log the internal error, but return a generic unauthorized message
			log.Printf("Error loading user data during login for %s: %v", payload.Username, err)
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		// 3. Check if user is active
		if !userInstance.Active {
			server.ErrorResponse(c, http.StatusUnauthorized, "User account is inactive")
			return
		}

		// 4. Verify password
		err = bcrypt.CompareHashAndPassword([]byte(userInstance.Password), []byte(payload.Password))
		if err != nil {
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		// --- Login Successful ---

		// 5. Update LastLogin timestamp
		nowUTC := time.Now().UTC()
		rfc3339Now := models.RFC3339Time(nowUTC) // Convert to custom type
		userInstance.LastLogin = &rfc3339Now     // Assign pointer to custom type

		// Save the updated user data
		if err := user.UpdateUser(userInstance); err != nil {
			log.Printf("Warning: Failed to update last_login for user %s: %v", userInstance.Username, err)
		}

		// 6. Generate JWT token
		jwtSecret := cfg.Auth.JWTSecret
		if jwtSecret == "" {
			log.Printf("Error: JWT secret not configured for session token generation.")
			server.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		expirationTime := nowUTC.Add(cfg.Auth.TokenDuration)
		issuedAt := nowUTC

		sessionID, err := generateSessionID()
		if err != nil {
			log.Printf("Error generating session ID for user %s: %v", userInstance.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate session identifier")
			return
		}

		claims := jwt.MapClaims{
			"aud":    AudienceWeb,
			"sub":    userInstance.ID,
			"name":   userInstance.Name,
			"jti":    sessionID,
			"scopes": userInstance.Scopes,
			"iss":    "PanelBase",
			"iat":    issuedAt.Unix(),
			"exp":    expirationTime.Unix(),
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		// 6b. Store session token info in tokenstore
		sessionInfo := tokenstore.TokenInfo{
			UserID:    userInstance.ID,
			Name:      "Web Session",
			Audience:  AudienceWeb,
			Scopes:    userInstance.Scopes,
			CreatedAt: models.RFC3339Time(issuedAt),       // Convert to custom type
			ExpiresAt: models.RFC3339Time(expirationTime), // Convert to custom type
		}
		if err := tokenstore.StoreToken(sessionID, sessionInfo); err != nil {
			log.Printf("Warning: Failed to store session token metadata for user %s (jti: %s): %v", userInstance.Username, sessionID, err)
		}

		// 7. Set cookie and return response
		c.SetCookie(cfg.Auth.CookieName, tokenString, int(cfg.Auth.TokenDuration.Seconds()), "/", "", false, true)
		server.SuccessResponse(c, "Login successful", gin.H{
			"token": tokenString,
		})
	}
}

// RefreshTokenHandler handles refreshing the JWT token and revokes the old one.
func RefreshTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract user information, token audience, and OLD JTI from context
		userIDVal, exists := c.Get(middleware.ContextKeyUserID)
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
			return
		}
		userID := userIDVal.(string)

		permissionsVal, exists := c.Get(middleware.ContextKeyScopes)
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "Permissions missing from context")
			return
		}
		userScopes := permissionsVal.(models.UserPermissions)

		tokenAudienceVal, exists := c.Get(middleware.ContextKeyAud)
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "Token audience missing from context")
			return
		}
		tokenAudience, ok := tokenAudienceVal.(jwt.ClaimStrings)
		if !ok {
			// Fallback check if it was somehow set as []string
			if audSlice, okSlice := tokenAudienceVal.([]string); okSlice {
				tokenAudience = jwt.ClaimStrings(audSlice)
				ok = true
			} else if audStr, okStr := tokenAudienceVal.(string); okStr {
				// Handle if it was set as single string
				tokenAudience = jwt.ClaimStrings{audStr}
				ok = true
			} else {
				server.ErrorResponse(c, http.StatusInternalServerError, "Invalid audience format in context")
				return
			}
		}

		// Get the OLD token's JTI
		oldJTIVal, exists := c.Get(middleware.ContextKeyJTI)
		if !exists {
			// If JTI is missing from the context (shouldn't happen if AuthMiddleware worked),
			// maybe log an error but potentially allow refresh? Or deny?
			// Let's deny for now, as JTI is needed for revocation.
			server.ErrorResponse(c, http.StatusUnauthorized, "Original token ID (jti) missing from context")
			return
		}
		oldTokenJTI, ok := oldJTIVal.(string)
		if !ok || oldTokenJTI == "" {
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid original token ID (jti) format in context")
			return
		}

		// 2. Check if the token audience is correct for refresh ('web')
		audienceValid := false
		for _, aud := range tokenAudience {
			if aud == AudienceWeb {
				audienceValid = true
				break
			}
		}
		if !audienceValid {
			server.ErrorResponse(c, http.StatusForbidden, "Token refresh requires a token with 'web' audience")
			return
		}

		// 3. Load user data to get Name
		userInstance, userExists, err := user.GetUserByID(userID)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data for refresh: "+err.Error())
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, "User associated with token not found")
			return
		}

		// 4. Generate a new JWT
		jwtSecret := cfg.Auth.JWTSecret
		if jwtSecret == "" {
			log.Printf("Error: JWT secret not configured for session token refresh.")
			server.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		newExpirationTime := time.Now().UTC().Add(cfg.Auth.TokenDuration) // Calculate based on UTC
		newIssuedAt := time.Now().UTC()                                   // Use UTC
		newSessionID, err := generateSessionID()
		if err != nil {
			log.Printf("Error generating new session ID for user %s: %v", userInstance.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate new session identifier")
			return
		}

		newClaims := jwt.MapClaims{
			"aud":    AudienceWeb, // Keep audience as web
			"sub":    userID,
			"name":   userInstance.Name, // Get name from loaded user
			"jti":    newSessionID,
			"scopes": userScopes, // Use scopes from the original token
			"iss":    "PanelBase",
			"iat":    newIssuedAt.Unix(),
			"exp":    newExpirationTime.Unix(),
		}

		newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
		newTokenString, err := newToken.SignedString([]byte(jwtSecret))
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate refreshed token")
			return
		}

		// 5. Store new session token info in tokenstore
		newSessionInfo := tokenstore.TokenInfo{
			UserID:    userID,
			Name:      "Web Session",
			Audience:  AudienceWeb,
			Scopes:    userScopes,
			CreatedAt: models.RFC3339Time(newIssuedAt),       // Convert to custom type
			ExpiresAt: models.RFC3339Time(newExpirationTime), // Convert to custom type
		}
		if err := tokenstore.StoreToken(newSessionID, newSessionInfo); err != nil {
			log.Printf("Warning: Failed to store refreshed session token metadata for user %s (jti: %s): %v", userInstance.Username, newSessionID, err)
		}

		// 6. Revoke the OLD session token
		if err := tokenstore.RevokeToken(oldTokenJTI); err != nil {
			log.Printf("Warning: Failed to revoke old session token (jti: %s) after refresh for user %s: %v", oldTokenJTI, userInstance.Username, err)
		}

		// 7. Set the new cookie and return the new token
		c.SetCookie(cfg.Auth.CookieName, newTokenString, int(cfg.Auth.TokenDuration.Seconds()), "/", "", false, true)
		server.SuccessResponse(c, "Token refreshed successfully", gin.H{
			"token": newTokenString,
		})
	}
}

// Add Audience constant if not already present
const AudienceWeb = "web"
