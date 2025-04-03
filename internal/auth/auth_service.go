package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config" // For JWTClaims, Audience constants
	"github.com/OG-Open-Source/PanelBase/internal/models" // For User structure
	"github.com/OG-Open-Source/PanelBase/internal/server" // For Response functions
	"github.com/OG-Open-Source/PanelBase/internal/user"   // Import the user service
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginPayload defines the structure for the login request body.
type LoginPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UsersData represents the structure of the users.json file.
// Duplicated here for simplicity, consider a shared model or user service.
type UsersData struct {
	JWTSecret string                 `json:"jwt_secret"`
	Users     map[string]models.User `json:"users"`
}

const usersFilePath = "configs/users.json"

// loadUsersData reads and parses the users.json file.
// TODO: Refactor this into a dedicated user service/repository.
func loadUsersData() (*UsersData, error) {
	data, err := os.ReadFile(usersFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	var usersData UsersData
	if err := json.Unmarshal(data, &usersData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal users data: %w", err)
	}
	return &usersData, nil
}

// findUserByUsername searches for a user by username in the loaded data.
// TODO: Refactor this into a dedicated user service/repository.
func findUserByUsername(username string, usersData *UsersData) (*models.User, string, error) {
	for userID, user := range usersData.Users {
		if user.Username == username {
			return &user, userID, nil
		}
	}
	return nil, "", fmt.Errorf("user not found")
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
		now := time.Now().UTC()
		userInstance.LastLogin = &now

		// Save the updated user data (including LastLogin) using the UserService
		if err := user.UpdateUser(userInstance); err != nil {
			log.Printf("Warning: Failed to update last_login for user %s: %v", userInstance.Username, err)
		}

		// 6. Generate JWT token with specified structure and order
		jwtSecret := cfg.Auth.JWTSecret
		if jwtSecret == "" {
			log.Printf("Error: JWT secret not configured for session token generation.")
			server.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		expirationTime := time.Now().Add(cfg.Auth.TokenDuration)
		issuedAt := time.Now()

		// Generate session ID
		sessionID, err := generateSessionID()
		if err != nil {
			log.Printf("Error generating session ID for user %s: %v", userInstance.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate session identifier")
			return
		}

		// Use jwt.MapClaims for specific order
		claims := jwt.MapClaims{
			"aud":    "web",                 // Audience: Web Session
			"sub":    userInstance.ID,       // Subject: User ID
			"name":   userInstance.Name,     // Name: User's display name
			"jti":    sessionID,             // JWT ID: Unique Session ID (ses_...)
			"scopes": userInstance.Scopes,   // Scopes: User's base permissions
			"iss":    "PanelBase",           // Issuer
			"iat":    issuedAt.Unix(),       // Issued At
			"exp":    expirationTime.Unix(), // Expiration Time
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		// 7. Set cookie and return response
		c.SetCookie(cfg.Auth.CookieName, tokenString, int(cfg.Auth.TokenDuration.Seconds()), "/", "", false, true)
		server.SuccessResponse(c, "Login successful", gin.H{
			"token": tokenString,
		})
	}
}

// RefreshTokenHandler handles refreshing the JWT token.
func RefreshTokenHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract user information and token audience from context (set by AuthMiddleware)
		userIDVal, exists := c.Get("userID")
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
			return
		}
		userID := userIDVal.(string)

		permissionsVal, exists := c.Get("userPermissions")
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "Permissions missing from context")
			return
		}
		userScopes := permissionsVal.(models.UserPermissions)

		tokenAudienceVal, exists := c.Get("tokenAudience")
		if !exists {
			server.ErrorResponse(c, http.StatusUnauthorized, "Token audience missing from context")
			return
		}
		// Ensure audience is correctly asserted (AuthMiddleware sets it as jwt.ClaimStrings)
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

		// 2. Check if the token audience is correct for refresh ('web')
		audienceValid := false
		for _, aud := range tokenAudience {
			if aud == "web" { // Check for the new audience value "web"
				audienceValid = true
				break
			}
		}
		if !audienceValid {
			server.ErrorResponse(c, http.StatusForbidden, "Token refresh requires a token with 'web' audience") // Updated error message
			return
		}

		// 3. Load user data to get Name (needed for new claim structure)
		// TODO: Optimize this - ideally AuthMiddleware could verify user existence
		// and maybe even attach the Name to context if needed frequently.
		userInstance, userExists, err := user.GetUserByID(userID)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data for refresh: "+err.Error())
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, "User associated with token not found")
			return
		}

		// 4. Generate a new JWT with updated expiration and potentially a new session ID (jti)
		jwtSecret := cfg.Auth.JWTSecret
		if jwtSecret == "" {
			log.Printf("Error: JWT secret not configured for session token generation.")
			server.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		expirationTime := time.Now().Add(cfg.Auth.TokenDuration)
		issuedAt := time.Now()
		newSessionID, err := generateSessionID() // Generate a new session ID for the refreshed token
		if err != nil {
			log.Printf("Error generating session ID for user %s: %v", userInstance.Username, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate session identifier")
			return
		}

		claims := jwt.MapClaims{
			"aud":    "web",
			"sub":    userID,
			"name":   userInstance.Name,
			"jti":    newSessionID,
			"scopes": userScopes,
			"iss":    "PanelBase",
			"iat":    issuedAt.Unix(),
			"exp":    expirationTime.Unix(),
		}

		newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		newTokenString, err := newToken.SignedString([]byte(jwtSecret))
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate refreshed token")
			return
		}

		// 5. Set the new token in the cookie and return success
		c.SetCookie(cfg.Auth.CookieName, newTokenString, int(cfg.Auth.TokenDuration.Seconds()), "/", "", false, true)
		server.SuccessResponse(c, "Token refreshed successfully", gin.H{
			"token": newTokenString,
		})
	}
}

// TODO: Remove findUserByUsername and loadUsersData once fully transitioned to userservice.
// These functions are likely redundant now.
var fileMutex sync.RWMutex // Keep mutex if these functions stay temporarily
