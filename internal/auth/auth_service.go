package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/middleware" // For JWTClaims, Audience constants
	"github.com/OG-Open-Source/PanelBase/internal/models"     // For User structure
	"github.com/OG-Open-Source/PanelBase/internal/server"     // For Response functions
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

// LoginHandler handles the user login request.
func LoginHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload LoginPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			server.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
			return
		}

		// Load user data (Consider caching or using a service)
		usersData, err := loadUsersData()
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data")
			fmt.Printf("Error loading user data: %v\n", err) // Log internal error
			return
		}

		// Find the user
		user, userID, err := findUserByUsername(payload.Username, usersData)
		if err != nil || (user != nil && !user.IsActive) {
			status := http.StatusUnauthorized
			message := "Invalid username or password"
			if err == nil && user != nil && !user.IsActive {
				message = "User account is inactive"
				status = http.StatusForbidden
			}
			server.ErrorResponse(c, status, message)
			return
		}

		// Compare the provided password with the stored hash
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password))
		if err != nil {
			// Password doesn't match
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		// --- Password matches --- Generate JWT for web session

		jwtSecret := []byte(usersData.JWTSecret) // Use secret from users.json
		// Session tokens expire in 7 days
		expirationTime := time.Now().Add(7 * 24 * time.Hour)

		claims := &middleware.JWTClaims{
			UserID:   userID, // Use the actual UserID from the map key
			Username: user.Username,
			JWTCustomClaims: middleware.JWTCustomClaims{
				Scopes: user.Scopes, // Include all user scopes for session token
			},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    "PanelBase",
				Audience:  jwt.ClaimStrings{middleware.AudienceWebSession},
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtSecret)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token")
			fmt.Printf("Error generating token: %v\n", err) // Log internal error
			return
		}

		// Set token in HTTP-only cookie
		c.SetCookie(
			cfg.Auth.CookieName,
			tokenString,
			int(7*24*time.Hour.Seconds()),    // maxAge in seconds
			"/",                              // path
			"",                               // domain (leave empty for current domain)
			cfg.Server.Mode != gin.DebugMode, // secure (true in release mode)
			true,                             // httpOnly
		)

		// Include the token in the success response data
		responseData := gin.H{
			"token": tokenString,
		}
		server.SuccessResponse(c, "Login successful", responseData)
	}
}
