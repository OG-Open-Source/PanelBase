package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config" // For JWTClaims, Audience constants
	logger "github.com/OG-Open-Source/PanelBase/internal/logging"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"  // Import middleware
	"github.com/OG-Open-Source/PanelBase/internal/models"      // For User structure
	"github.com/OG-Open-Source/PanelBase/internal/server"      // For Response functions
	"github.com/OG-Open-Source/PanelBase/internal/token_store" // Import token store
	"github.com/OG-Open-Source/PanelBase/internal/user"        // Import the user service
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
			logger.ErrorPrintf("AUTH", "LOGIN_FIND_USER", "Error retrieving user %s: %v", payload.Username, err)
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
			logger.ErrorPrintf("AUTH", "LOGIN_UPDATE_USER", "Warning: Failed to update last_login for user %s: %v", userInstance.Username, err)
		}

		// 6. Generate JWT token
		jwtSecret := cfg.Auth.JWTSecret
		if jwtSecret == "" {
			logger.ErrorPrintf("AUTH", "LOGIN_JWT_SECRET", "Error: JWT secret not configured for session token generation.")
			server.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		expirationTime := nowUTC.Add(cfg.Auth.TokenDuration)
		issuedAt := nowUTC

		sessionID, err := generateSessionID()
		if err != nil {
			logger.ErrorPrintf("AUTH", "LOGIN_SESSION_ID", "Error generating session ID for user %s: %v", payload.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate session identifier")
			return
		}

		claims := jwt.MapClaims{
			"aud":    middleware.AudienceWeb,
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
			logger.ErrorPrintf("AUTH", "LOGIN_SIGN_TOKEN", "Error signing token for user %s: %v", payload.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		// 6b. Store session token info in token_store
		sessionInfo := token_store.TokenInfo{
			UserID:    userInstance.ID,
			Name:      "Web Session",
			Audience:  middleware.AudienceWeb,
			Scopes:    userInstance.Scopes,
			CreatedAt: models.RFC3339Time(issuedAt),       // Convert to custom type
			ExpiresAt: models.RFC3339Time(expirationTime), // Convert to custom type
		}
		if err := token_store.StoreToken(sessionID, sessionInfo); err != nil {
			logger.ErrorPrintf("AUTH", "LOGIN_STORE_TOKEN", "Warning: Failed to store session token metadata for user %s (jti: %s): %v", payload.Username, sessionID, err)
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
		userIDVal, exists := c.Get(string(middleware.ContextKeyUserID))
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
			return
		}
		userID := userIDVal.(string)

		permissionsVal, exists := c.Get(string(middleware.ContextKeyPermissions))
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "Permissions missing from context")
			return
		}
		userScopes := permissionsVal.(models.UserPermissions)

		tokenAudienceVal, exists := c.Get(string(middleware.ContextKeyAudience))
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "Token audience missing from context")
			return
		}
		tokenAudience := tokenAudienceVal.(string)

		// Get the OLD token's JTI
		oldJTIVal, exists := c.Get(string(middleware.ContextKeyJTI))
		if !exists {
			// If JTI is missing from the context (shouldn't happen if AuthMiddleware worked),
			// maybe log an error but potentially allow refresh? Or deny?
			// Let's deny for now, as JTI is needed for revocation.
			server.ErrorResponse(c, http.StatusUnauthorized, "Original token ID (jti) missing from context")
			return
		}
		oldTokenJTI := oldJTIVal.(string)

		// 2. Check if the token audience is correct for refresh ('web')
		if tokenAudience != middleware.AudienceWeb {
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
			logger.ErrorPrintf("AUTH", "REFRESH_JWT_SECRET", "Error: JWT secret not configured for session token refresh.")
			server.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		newExpirationTime := time.Now().UTC().Add(cfg.Auth.TokenDuration) // Calculate based on UTC
		newIssuedAt := time.Now().UTC()                                   // Use UTC
		newSessionID, err := generateSessionID()
		if err != nil {
			logger.ErrorPrintf("AUTH", "REFRESH_SESSION_ID", "Error generating new session ID for user %s: %v", userInstance.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate new session identifier")
			return
		}

		newClaims := jwt.MapClaims{
			"aud":    middleware.AudienceWeb, // Keep audience as web
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
			logger.ErrorPrintf("AUTH", "REFRESH_SIGN_TOKEN", "Failed to sign refreshed token for user %s: %v", userInstance.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate refreshed token")
			return
		}

		// 5. Store new session token info in token_store
		newSessionInfo := token_store.TokenInfo{
			UserID:    userID,
			Name:      "Web Session",
			Audience:  middleware.AudienceWeb,
			Scopes:    userScopes,
			CreatedAt: models.RFC3339Time(newIssuedAt),       // Convert to custom type
			ExpiresAt: models.RFC3339Time(newExpirationTime), // Convert to custom type
		}
		if err := token_store.StoreToken(newSessionID, newSessionInfo); err != nil {
			logger.ErrorPrintf("AUTH", "REFRESH_STORE_TOKEN", "Warning: Failed to store refreshed session token metadata for user %s (jti: %s): %v", userInstance.Username, newSessionID, err)
		}

		// 6. Revoke the OLD session token
		if err := token_store.RevokeToken(oldTokenJTI); err != nil {
			logger.ErrorPrintf("AUTH", "REFRESH_REVOKE", "Failed to revoke old token %s during refresh: %v", oldTokenJTI, err)
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

// RegisterPayload defines the structure for the registration request body.
// Assuming this is defined in models package, otherwise define here.
type RegisterPayload struct { // If not in models, define here
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"` // Removed binding:"required"
	Name     string `json:"name"`
}

// RegisterHandler handles new user registration.
func RegisterHandler(c *gin.Context) {
	var payload RegisterPayload // Use local or models.RegisterPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Basic validation (Email is now optional)
	if payload.Username == "" || payload.Password == "" {
		server.ErrorResponse(c, http.StatusBadRequest, "Username and password are required")
		return
	}

	// Check if username already exists
	_, exists, err := user.GetUserByUsername(payload.Username)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Error checking username availability: "+err.Error())
		return
	}
	if exists {
		server.ErrorResponse(c, http.StatusConflict, "Username already exists")
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.ErrorPrintf("AUTH", "REGISTER_HASH", "Failed to hash password for user %s: %v", payload.Username, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Generate user ID
	userID, err := generateUserID()
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate user ID: "+err.Error())
		return
	}

	// Generate user-specific JWT secret
	userJwtSecret, err := generateRandomString(32)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate JWT secret: "+err.Error())
		return
	}

	// Determine user's name
	userName := payload.Name
	if userName == "" {
		userName = payload.Username
	}

	// Define default scopes for new users (MODIFIED HERE)
	defaultScopes := models.UserPermissions{
		"account": {"read", "update", "delete"},
		"api":     {"read:list", "read:item", "create", "update", "delete"},
		// No other scopes assigned by default
	}

	// Create the new user object
	newUser := models.User{
		ID:        userID,
		Username:  payload.Username,
		Password:  string(hashedPassword),
		Name:      userName,
		Email:     payload.Email,
		CreatedAt: models.RFC3339Time(time.Now().UTC()),
		Active:    true,
		Scopes:    defaultScopes, // Assign the restricted default scopes
		API: models.UserAPISettings{
			JwtSecret: userJwtSecret,
		},
	}

	// Save the new user
	if err := user.AddUser(newUser); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			server.ErrorResponse(c, http.StatusConflict, err.Error()) // Username or ID conflict
		} else {
			logger.ErrorPrintf("AUTH", "REGISTER_ADD_USER", "Failed to add user %s: %v", payload.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to register user")
		}
		return
	}

	logger.Printf("AUTH", "REGISTER", "User registered successfully: %s (ID: %s)", payload.Username, payload.Username)
	// Create response map, explicitly omitting sensitive data like password and secrets
	responseMap := map[string]interface{}{
		"id":         newUser.ID,
		"username":   newUser.Username,
		"name":       newUser.Name,
		"email":      newUser.Email,
		"created_at": newUser.CreatedAt, // Assuming RFC3339Time marshals correctly
		"active":     newUser.Active,
		"scopes":     newUser.Scopes,
		// "password" key is omitted
		// "api" key (containing jwt_secret) is omitted
	}

	server.SuccessResponse(c, "User registered successfully", responseMap)
}

// generateUserID creates a unique user identifier.
func generateUserID() (string, error) {
	bytesLength := 8
	b := make([]byte, bytesLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes for user ID: %w", err)
	}
	return "usr_" + hex.EncodeToString(b), nil
}

// generateRandomString generates a secure random hex string.
// (Assuming this function exists or is added)
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
