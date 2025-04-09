package auth

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/pkg/authutils" // UPDATED PATH for generators
	"github.com/OG-Open-Source/PanelBase/pkg/models"
	"github.com/OG-Open-Source/PanelBase/pkg/serverutils"
	"github.com/OG-Open-Source/PanelBase/pkg/tokenstore"
	"github.com/OG-Open-Source/PanelBase/pkg/userservice"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginHandler handles user login requests.
func LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload models.LoginPayload // Use LoginPayload from pkg/models
		if err := c.ShouldBindJSON(&payload); err != nil {
			serverutils.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// 1. Load user data using the user service
		userInstance, userExists, err := userservice.GetUserByUsername(payload.Username)
		if err != nil {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}
		if !userExists {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		// 3. Check if user is active
		if !userInstance.Active {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "User account is inactive")
			return
		}

		// 4. Verify password
		err = bcrypt.CompareHashAndPassword([]byte(userInstance.Password), []byte(payload.Password))
		if err != nil {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		// --- Login Successful ---

		// 5. Update LastLogin timestamp
		nowUTC := time.Now().UTC()
		rfc3339Now := models.RFC3339Time(nowUTC) // Convert to custom type
		userInstance.LastLogin = &rfc3339Now     // Assign pointer to custom type

		// Save the updated user data
		if err := userservice.UpdateUser(userInstance); err != nil {
			log.Printf("%s Warning: Failed to update last_login for user %s: %v", nowUTC.Format(time.RFC3339), userInstance.Username, err)
		}

		// 6. Generate JWT token
		if userInstance.API.JwtSecret == "" {
			log.Printf("%s Error: JWT secret not configured for session token generation.", nowUTC.Format(time.RFC3339))
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		expirationTime := nowUTC.Add(24 * time.Hour)
		issuedAt := nowUTC

		sessionID, err := authutils.GenerateSessionID() // Use generator from pkg/authutils
		if err != nil {
			log.Printf("%s Error generating session ID for user %s: %v", nowUTC.Format(time.RFC3339), userInstance.Username, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate session identifier")
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
		tokenString, err := token.SignedString([]byte(userInstance.API.JwtSecret))
		if err != nil {
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		// 6b. Store session token info in token_store
		sessionInfo := tokenstore.TokenInfo{
			UserID:    userInstance.ID,
			Name:      "Web Session",
			Audience:  middleware.AudienceWeb,
			Scopes:    userInstance.Scopes,
			CreatedAt: models.RFC3339Time(issuedAt),       // Convert to custom type
			ExpiresAt: models.RFC3339Time(expirationTime), // Convert to custom type
		}
		if err := tokenstore.StoreToken(sessionID, sessionInfo); err != nil {
			log.Printf("%s Warning: Failed to store session token metadata for user %s (jti: %s): %v", nowUTC.Format(time.RFC3339), userInstance.Username, sessionID, err)
		}

		// 7. Set cookie and return response
		cookieName := "panelbase_jwt"                            // Use fixed name or get from env/default
		cookieDurationSeconds := int((24 * time.Hour).Seconds()) // Use fixed duration or get from env/default
		c.SetCookie(cookieName, tokenString, cookieDurationSeconds, "/", "", false, true)
		serverutils.SuccessResponse(c, "Login successful", gin.H{
			"token": tokenString,
		})
	}
}

// RefreshTokenHandler handles refreshing the JWT token and revokes the old one.
func RefreshTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract user information, token audience, and OLD JTI from context
		userIDVal, exists := c.Get(string(middleware.ContextKeyUserID))
		if !exists {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
			return
		}
		userID := userIDVal.(string)

		permissionsVal, exists := c.Get(string(middleware.ContextKeyPermissions))
		if !exists {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Permissions missing from context")
			return
		}
		userScopes := permissionsVal.(models.UserPermissions)

		tokenAudienceVal, exists := c.Get(string(middleware.ContextKeyAudience))
		if !exists {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Token audience missing from context")
			return
		}
		tokenAudience := tokenAudienceVal.(string)

		// Get the OLD token's JTI
		oldJTIVal, exists := c.Get(string(middleware.ContextKeyJTI))
		if !exists {
			// If JTI is missing from the context (shouldn't happen if AuthMiddleware worked),
			// maybe log an error but potentially allow refresh? Or deny?
			// Let's deny for now, as JTI is needed for revocation.
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Original token ID (jti) missing from context")
			return
		}
		oldTokenJTI := oldJTIVal.(string)

		// 2. Check if the token audience is correct for refresh ('web')
		if tokenAudience != middleware.AudienceWeb {
			serverutils.ErrorResponse(c, http.StatusForbidden, "Token refresh requires a token with 'web' audience")
			return
		}

		// 3. Load user data to get Name
		userInstance, userExists, err := userservice.GetUserByID(userID)
		if err != nil {
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data for refresh: "+err.Error())
			return
		}
		if !userExists {
			serverutils.ErrorResponse(c, http.StatusNotFound, "User associated with token not found")
			return
		}

		// 4. Generate a new JWT
		if userInstance.API.JwtSecret == "" {
			log.Printf("%s Error: JWT secret not configured for session token refresh.", time.Now().UTC().Format(time.RFC3339))
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Session token configuration error")
			return
		}

		newExpirationTime := time.Now().UTC().Add(24 * time.Hour)
		newIssuedAt := time.Now().UTC()
		newSessionID, err := authutils.GenerateSessionID() // Use generator from pkg/authutils
		if err != nil {
			log.Printf("%s Error generating new session ID for user %s: %v", time.Now().UTC().Format(time.RFC3339), userInstance.Username, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate new session identifier")
			return
		}

		newClaims := jwt.MapClaims{
			"aud":    middleware.AudienceWeb,
			"sub":    userID,
			"name":   userInstance.Name,
			"jti":    newSessionID,
			"scopes": userScopes,
			"iss":    "PanelBase",
			"iat":    newIssuedAt.Unix(),
			"exp":    newExpirationTime.Unix(),
		}

		newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
		newTokenString, err := newToken.SignedString([]byte(userInstance.API.JwtSecret))
		if err != nil {
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate refreshed token")
			return
		}

		// 5. Store new session token info in token_store
		newSessionInfo := tokenstore.TokenInfo{
			UserID:    userID,
			Name:      "Web Session",
			Audience:  middleware.AudienceWeb,
			Scopes:    userScopes,
			CreatedAt: models.RFC3339Time(newIssuedAt),
			ExpiresAt: models.RFC3339Time(newExpirationTime),
		}
		if err := tokenstore.StoreToken(newSessionID, newSessionInfo); err != nil {
			log.Printf("%s Warning: Failed to store refreshed session token metadata for user %s (jti: %s): %v", time.Now().UTC().Format(time.RFC3339), userInstance.Username, newSessionID, err)
		}

		// 6. Revoke the OLD session token
		if err := tokenstore.RevokeToken(oldTokenJTI); err != nil {
			log.Printf("%s Warning: Failed to revoke old session token (jti: %s) after refresh for user %s: %v", time.Now().UTC().Format(time.RFC3339), oldTokenJTI, userInstance.Username, err)
		}

		// 7. Set the new cookie and return the new token
		cookieName := "panelbase_jwt"                            // Use fixed name or get from env/default
		cookieDurationSeconds := int((24 * time.Hour).Seconds()) // Use fixed duration or get from env/default
		c.SetCookie(cookieName, newTokenString, cookieDurationSeconds, "/", "", false, true)
		serverutils.SuccessResponse(c, "Token refreshed successfully", gin.H{
			"token": newTokenString,
		})
	}
}

// RegisterHandler handles new user registration requests.
func RegisterHandler(c *gin.Context) {
	var payload models.RegisterPayload // Use RegisterPayload from pkg/models
	if err := c.ShouldBindJSON(&payload); err != nil {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Validate input (e.g., password complexity if needed)

	// Check if username already exists
	_, exists, err := userservice.GetUserByUsername(payload.Username)
	if err != nil {
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Error checking username availability")
		return
	}
	if exists {
		serverutils.ErrorResponse(c, http.StatusConflict, "Username already taken")
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Generate a unique user ID
	userID, err := authutils.GenerateUserID() // Use generator from pkg/authutils
	if err != nil {
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate user ID")
		return
	}

	// Generate a secure random JWT secret for the user
	jwtSecretBytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(jwtSecretBytes); err != nil {
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate JWT secret")
		return
	}
	jwtSecret := hex.EncodeToString(jwtSecretBytes)

	// Create the new user object
	newUser := models.User{
		ID:        userID,
		Username:  payload.Username,
		Password:  string(hashedPassword),
		Email:     payload.Email, // Now optional
		Name:      payload.Name,
		Active:    true,                                     // Activate user immediately
		Scopes:    map[string][]string{"default": {"read"}}, // Assign default minimal scope
		CreatedAt: models.RFC3339Time(time.Now().UTC()),
		LastLogin: nil,
		API: models.UserAPISettings{
			JwtSecret: jwtSecret, // Assign the generated secret
			// Tokens map removed
		},
	}

	// Use the UserService to create the user (Corrected function name)
	if err := userservice.AddUser(newUser); err != nil {
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to register user: "+err.Error())
		return
	}

	// Return success response (excluding sensitive info)
	responseUser := map[string]interface{}{
		"id":         newUser.ID,
		"username":   newUser.Username,
		"email":      newUser.Email,
		"name":       newUser.Name,
		"active":     newUser.Active,
		"scopes":     newUser.Scopes,
		"created_at": newUser.CreatedAt,
		"last_login": newUser.LastLogin,
		// Explicitly exclude password and api settings
	}

	serverutils.SuccessResponse(c, "User registered successfully", responseUser)
}
