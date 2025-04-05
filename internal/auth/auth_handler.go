package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/gin-gonic/gin"
)

// RegisterRequest defines the structure for the user registration request body.
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name,omitempty"`
}

// RegisterHandler handles the user registration request.
func RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	// Bind and validate request body
	if err := c.ShouldBindJSON(&req); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Call the registration service function
	newUser, err := RegisterUser(req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "already taken") {
			statusCode = http.StatusConflict // Use 409 Conflict for existing username
		}
		server.ErrorResponse(c, statusCode, err.Error())
		return
	}

	// Return success response (don't return password hash)
	// Consider what information is useful to return upon successful registration
	response := gin.H{
		"id":         newUser.ID,
		"username":   newUser.Username,
		"email":      newUser.Email,
		"name":       newUser.Name,
		"created_at": newUser.CreatedAt.Time().Format(time.RFC3339),
		"active":     newUser.Active,
	}
	server.SuccessResponse(c, "User registered successfully", response)
}

// LoginHandler handles user login requests.
// ... existing LoginHandler code ...

// RefreshTokenHandler handles token refresh requests.
// ... existing RefreshTokenHandler code ...
 