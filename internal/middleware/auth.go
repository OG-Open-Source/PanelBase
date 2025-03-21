package middleware

import (
	"net/http"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/labstack/echo/v4"
)

// AuthMiddleware handles authentication for both JWT tokens and API keys
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip authentication for public endpoints
			path := c.Request().URL.Path
			if isPublicPath(path) {
				return next(c)
			}

			// Get Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authorization header required",
				})
			}

			var userInfo *user.User
			var err error

			// Check if it's a Bearer token or API key
			if strings.HasPrefix(authHeader, "Bearer ") {
				// It's a Bearer token, use JWT verification
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := user.VerifyJWT(tokenString)
				if err != nil {
					logger.Log.Warnf("Invalid JWT token: %v", err)
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Invalid token",
					})
				}

				// Get username from claims
				username, ok := claims["username"].(string)
				if !ok {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Invalid token content",
					})
				}

				// Get user info
				userInfo, err = user.GetUser(username)
				if err != nil {
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "User not found",
					})
				}
			} else if strings.HasPrefix(authHeader, "ApiKey ") {
				// It's an API key
				apiKeyString := strings.TrimPrefix(authHeader, "ApiKey ")
				userInfo, err = user.VerifyAPIKey(apiKeyString)
				if err != nil {
					logger.Log.Warnf("Invalid API key: %v", err)
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Invalid API key",
					})
				}
			} else {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid authorization format",
				})
			}

			// Set user info in context
			c.Set("user", userInfo)
			return next(c)
		}
	}
}

// AdminMiddleware checks if the user is an admin
func AdminMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userInfo, ok := c.Get("user").(*user.User)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Not authenticated",
				})
			}

			if userInfo.Role != "admin" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Admin permission required",
				})
			}

			return next(c)
		}
	}
}

// isPublicPath checks if the path is a public path
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/login",
		"/health",
		"/favicon.ico",
		"/index.html",
	}

	for _, publicPath := range publicPaths {
		if strings.HasSuffix(path, publicPath) {
			return true
		}
	}

	// Static files
	if strings.Contains(path, "/static/") {
		return true
	}

	return false
}
