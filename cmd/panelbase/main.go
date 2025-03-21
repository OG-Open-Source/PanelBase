package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	customMiddleware "github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/theme"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Template renderer for Echo
type TemplateRenderer struct {
	templates *template.Template
}

// Render implements the echo.Renderer interface
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// LayoutData contains the data to be passed to the layout template
type LayoutData struct {
	EntryPoint string
}

// ThemeInstallRequest represents the request to install a theme
type ThemeInstallRequest struct {
	ThemeURL string `json:"theme_url"`
}

// APIKeyResponse represents the response for an API key request
type APIKeyResponse struct {
	APIKey   string    `json:"api_key"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	Expires  time.Time `json:"expires_at"`
}

// UserListItem represents a user in the user list response
type UserListItem struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	LastLogin time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
	Role     string `json:"role" validate:"required"`
	IsActive bool   `json:"is_active"`
}

// UpdateUserStatusRequest represents a request to update a user's active status
type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// downloadFile downloads a file from the given URL to the given path
func downloadFile(url, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// installTheme installs a theme from a JSON configuration file
func installTheme(themeConfigPath, webDir string) (*theme.Theme, error) {
	// Read theme configuration
	data, err := ioutil.ReadFile(themeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error reading theme file: %v", err)
	}

	var themeConfig theme.ThemeConfig
	if err := json.Unmarshal(data, &themeConfig); err != nil {
		return nil, fmt.Errorf("error parsing theme file: %v", err)
	}

	// Create theme directory
	themeDir := filepath.Join(webDir, themeConfig.Theme.Directory)
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating theme directory: %v", err)
	}

	// Download and install theme files
	for path, content := range themeConfig.Theme.Structure {
		switch v := content.(type) {
		case string:
			// It's a direct URL, download the file
			filePath := filepath.Join(themeDir, path)
			// Create directory if needed
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return nil, fmt.Errorf("error creating directory for %s: %v", path, err)
			}
			if err := downloadFile(v, filePath); err != nil {
				return nil, fmt.Errorf("error downloading file %s: %v", path, err)
			}
		case map[string]interface{}:
			// It's a directory with nested files
			for subPath, subContent := range v {
				if subURL, ok := subContent.(string); ok {
					dirPath := filepath.Join(themeDir, path)
					if err := os.MkdirAll(dirPath, 0755); err != nil {
						return nil, fmt.Errorf("error creating directory %s: %v", dirPath, err)
					}
					filePath := filepath.Join(dirPath, subPath)
					if err := downloadFile(subURL, filePath); err != nil {
						return nil, fmt.Errorf("error downloading file %s: %v", subPath, err)
					}
				}
			}
		}
	}

	return &themeConfig.Theme, nil
}

// checkAdminPermission checks if the user has admin permission
func checkAdminPermission(c echo.Context) bool {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// Bearer token format
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false
	}

	// Extract token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Verify JWT token
	claims, err := user.VerifyJWT(tokenString)
	if err != nil {
		logger.Log.Warnf("Invalid JWT token: %v", err)
		return false
	}

	// Check if user is admin
	_, ok := claims["username"].(string)
	if !ok {
		return false
	}

	role, ok := claims["role"].(string)
	if !ok {
		return false
	}

	return role == "admin"
}

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.InitLogger(); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	// Initialize user management
	cfg := config.GetConfig()
	userConfigPath := filepath.Join("configs", "users.json")
	jwtSecret := cfg.Security.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-change-in-production" // Default secret
		logger.Log.Warn("Using default JWT secret. Change this in production!")
	}

	if err := user.Init(userConfigPath, jwtSecret); err != nil {
		logger.Log.Warnf("Failed to initialize user store: %v", err)
		// Continue anyway, we'll create default users later
	}

	// Load theme
	themeFilePath := cfg.UI.ThemeFile
	if _, err := theme.LoadTheme(themeFilePath); err != nil {
		logger.Log.Warnf("Failed to load theme: %v", err)
	}

	// Create Echo instance
	e := echo.New()

	// Define the layout template
	layoutTemplate := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PanelBase</title>
    <style>
        /* Reset CSS */
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: #f5f5f5;
            display: flex;
            flex-direction: column;
            height: 100vh;
            overflow: hidden;
        }
        
        /* Title bar styles */
        .title-bar {
            background-color: #2c3e50;
            color: white;
            padding: 10px 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
            z-index: 100;
        }
        
        .title-bar .left {
            display: flex;
            align-items: center;
        }
        
        .title-bar .logo {
            font-size: 1.5rem;
            font-weight: bold;
            margin-right: 20px;
        }
        
        .title-bar .theme-info {
            font-size: 1rem;
        }
        
        .title-bar .right {
            display: flex;
            align-items: center;
        }
        
        .title-bar .user-info {
            display: flex;
            align-items: center;
        }
        
        .title-bar .user-avatar {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            background-color: #3498db;
            display: flex;
            justify-content: center;
            align-items: center;
            margin-right: 10px;
            font-weight: bold;
        }
        
        .title-bar .username {
            margin-right: 15px;
        }
        
        .title-bar .btn {
            background-color: #e74c3c;
            color: white;
            border: none;
            padding: 5px 10px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 0.9rem;
            margin-left: 5px;
        }
        
        .title-bar .logout-btn {
            background-color: #e74c3c;
        }
        
        .title-bar .logout-btn:hover {
            background-color: #c0392b;
        }
        
        .title-bar .apikey-btn {
            background-color: #2980b9;
        }
        
        .title-bar .apikey-btn:hover {
            background-color: #216a9e;
        }
        
        /* Modal styles */
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.5);
            z-index: 200;
            justify-content: center;
            align-items: center;
        }
        
        .modal-content {
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
            width: 500px;
            max-width: 90%;
        }
        
        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }
        
        .modal-title {
            font-size: 1.2rem;
            font-weight: bold;
        }
        
        .close-btn {
            font-size: 1.5rem;
            cursor: pointer;
            background: none;
            border: none;
        }
        
        .modal-body {
            margin-bottom: 15px;
        }
        
        .api-key-container {
            background-color: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            margin-bottom: 10px;
            word-break: break-all;
            font-family: monospace;
        }
        
        .copy-btn {
            background-color: #3498db;
            color: white;
            border: none;
            padding: 5px 10px;
            border-radius: 4px;
            cursor: pointer;
        }
        
        .copy-btn:hover {
            background-color: #2980b9;
        }
        
        /* Content frame styles */
        .content-frame {
            flex: 1;
            border: none;
            width: 100%;
        }
    </style>
</head>
<body>
    <!-- Title Bar -->
    <div class="title-bar">
        <div class="left">
            <div class="logo">PanelBase</div>
            <div class="theme-info" id="theme-info">Theme: Loading...</div>
        </div>
        <div class="right">
            <div class="user-info">
                <div class="user-avatar" id="user-avatar">U</div>
                <div class="username" id="username">Loading...</div>
                <button class="btn apikey-btn" id="apikey-btn">API Key</button>
                <button class="btn logout-btn" id="logout-btn">Logout</button>
            </div>
        </div>
    </div>
    
    <!-- Content Frame -->
    <iframe id="content-frame" class="content-frame" src="about:blank"></iframe>
    
    <!-- API Key Modal -->
    <div id="apikey-modal" class="modal">
        <div class="modal-content">
            <div class="modal-header">
                <div class="modal-title">Your API Key</div>
                <button class="close-btn" id="close-modal">&times;</button>
            </div>
            <div class="modal-body">
                <p>Your API key will expire in 30 days. Keep it secure!</p>
                <div class="api-key-container" id="api-key">Generating...</div>
                <p>To use this API key in requests, include it in the Authorization header:</p>
                <pre>Authorization: ApiKey YOUR_API_KEY</pre>
            </div>
            <button class="copy-btn" id="copy-key">Copy API Key</button>
        </div>
    </div>
    
    <script>
        document.addEventListener('DOMContentLoaded', () => {
            // Check if token exists, if not redirect to login page
            const token = localStorage.getItem('token');
            if (!token) {
                // Token doesn't exist, user is not logged in
                window.location.href = '/{{.EntryPoint}}/index.html';
                return;
            }
            
            // Get the entry point from the URL
            const entryPoint = '{{.EntryPoint}}';
            
            // Fetch the current theme and user information
            const fetchThemeAndUser = async () => {
                try {
                    // Fetch theme info with authentication
                    const themeResponse = await fetch('/' + entryPoint + '/api/theme', {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    
                    if (!themeResponse.ok) {
                        if (themeResponse.status === 401) {
                            // Unauthorized, token might be expired
                            localStorage.removeItem('token');
                            window.location.href = '/' + entryPoint + '/index.html';
                            return;
                        }
                        throw new Error('Failed to fetch theme info');
                    }
                    
                    const themeInfo = await themeResponse.json();
                    
                    // Fetch user info
                    const userResponse = await fetch('/' + entryPoint + '/api/user', {
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    
                    if (!userResponse.ok) {
                        if (userResponse.status === 401) {
                            // Unauthorized, token might be expired
                            localStorage.removeItem('token');
                            window.location.href = '/' + entryPoint + '/index.html';
                            return;
                        }
                        throw new Error('Failed to fetch user info');
                    }
                    
                    const userInfo = await userResponse.json();
                    
                    // Update the UI
                    document.getElementById('theme-info').textContent = 'Theme: ' + themeInfo.name;
                    document.getElementById('username').textContent = userInfo.username;
                    document.getElementById('user-avatar').textContent = userInfo.username.charAt(0).toUpperCase();
                    
                    // Set the content frame src to the theme's index.html
                    document.getElementById('content-frame').src = '/' + entryPoint + '/' + themeInfo.directory + '/index.html';
                    
                    // Set document title
                    document.title = 'PanelBase - ' + themeInfo.name;
                } catch (error) {
                    console.error('Error:', error);
                    // If we can't fetch the info, redirect to login
                    localStorage.removeItem('token');
                    window.location.href = '/' + entryPoint + '/index.html';
                }
            };
            
            // API Key modal functionality
            const apiKeyModal = document.getElementById('apikey-modal');
            const apiKeyBtn = document.getElementById('apikey-btn');
            const closeModal = document.getElementById('close-modal');
            const copyKeyBtn = document.getElementById('copy-key');
            
            apiKeyBtn.addEventListener('click', async () => {
                apiKeyModal.style.display = 'flex';
                
                try {
                    // Request a new API key
                    const response = await fetch('/' + entryPoint + '/api/apikey', {
                        method: 'POST',
                        headers: {
                            'Authorization': 'Bearer ' + token
                        }
                    });
                    
                    if (!response.ok) {
                        throw new Error('Failed to generate API key');
                    }
                    
                    const data = await response.json();
                    document.getElementById('api-key').textContent = data.api_key;
                } catch (error) {
                    console.error('Error:', error);
                    document.getElementById('api-key').textContent = 'Error generating API key';
                }
            });
            
            closeModal.addEventListener('click', () => {
                apiKeyModal.style.display = 'none';
            });
            
            // Close modal when clicking outside
            window.addEventListener('click', (event) => {
                if (event.target === apiKeyModal) {
                    apiKeyModal.style.display = 'none';
                }
            });
            
            // Copy API key to clipboard
            copyKeyBtn.addEventListener('click', () => {
                const apiKeyText = document.getElementById('api-key').textContent;
                navigator.clipboard.writeText(apiKeyText).then(() => {
                    alert('API key copied to clipboard!');
                }).catch(err => {
                    console.error('Could not copy text: ', err);
                });
            });
            
            // Logout button functionality
            document.getElementById('logout-btn').addEventListener('click', () => {
                localStorage.removeItem('token');
                window.location.href = '/' + entryPoint + '/index.html';
            });
            
            // Initialize the page
            fetchThemeAndUser();
        });
    </script>
</body>
</html>
`

	// Set up template renderer
	t := &TemplateRenderer{
		templates: template.Must(template.New("layout").Parse(layoutTemplate)),
	}
	e.Renderer = t

	// Core middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		logger.Log.Fatal(err)
	}

	// Get entry point
	entryPoint := cfg.Server.Entry

	// Get web directory path
	webDir := filepath.Join(wd, "web")

	// Check if web directory exists
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		logger.Log.Fatalf("Web directory not found: %s", webDir)
	}

	// Create theme directory if needed
	if currentTheme := theme.GetCurrentTheme(); currentTheme != nil {
		themeDir, err := theme.EnsureThemeDirectory(webDir)
		if err != nil {
			logger.Log.Warnf("Failed to ensure theme directory: %v", err)
		} else {
			logger.Log.Infof("Using theme: %s (%s)", currentTheme.Name, themeDir)
		}
	}

	// Handle the entry point to serve layout template
	e.GET(fmt.Sprintf("/%s", entryPoint), func(c echo.Context) error {
		return c.Render(http.StatusOK, "layout", LayoutData{
			EntryPoint: entryPoint,
		})
	})

	// API routes
	api := e.Group(fmt.Sprintf("/%s/api", entryPoint))

	// Public API endpoints (no auth required)
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// Login endpoint with JWT token generation
	api.POST("/login", func(c echo.Context) error {
		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		req := new(LoginRequest)
		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"message": "Invalid request",
			})
		}

		// Authenticate user with our new user management
		authenticatedUser, err := user.Authenticate(req.Username, req.Password)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"message": "Invalid credentials",
			})
		}

		// Generate JWT token with 24 hour expiry
		token, err := user.GenerateJWT(authenticatedUser.Username, 24)
		if err != nil {
			logger.Log.Errorf("Failed to generate JWT: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"message": "Authentication error",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"token":    token,
			"message":  "Login successful",
			"username": authenticatedUser.Username,
			"role":     authenticatedUser.Role,
		})
	})

	// Protected API endpoints (require authentication)
	auth := api.Group("")
	auth.Use(customMiddleware.AuthMiddleware())

	// User info endpoint with JWT verification
	auth.GET("/user", func(c echo.Context) error {
		userInfo, ok := c.Get("user").(*user.User)
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get user information",
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"username": userInfo.Username,
			"role":     userInfo.Role,
		})
	})

	// Theme info endpoint
	auth.GET("/theme", func(c echo.Context) error {
		currentTheme := theme.GetCurrentTheme()
		if currentTheme == nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "No theme loaded",
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"name":        currentTheme.Name,
			"authors":     currentTheme.Authors,
			"version":     currentTheme.Version,
			"description": currentTheme.Description,
			"directory":   currentTheme.Directory,
		})
	})

	// Generate API key endpoint (requires authentication)
	auth.POST("/apikey", func(c echo.Context) error {
		userInfo, ok := c.Get("user").(*user.User)
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get user information",
			})
		}

		// Generate API key with 30 days expiry
		apiKey, err := user.GenerateAPIKey(userInfo.Username, 30)
		if err != nil {
			logger.Log.Errorf("Failed to generate API key: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to generate API key",
			})
		}

		// Get updated user data for response
		userData, err := user.GetUser(userInfo.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "User not found",
			})
		}

		return c.JSON(http.StatusOK, APIKeyResponse{
			APIKey:   apiKey,
			Username: userData.Username,
			Role:     userData.Role,
			Expires:  userData.APIKeyExpiry,
		})
	})

	// Admin-only endpoints (require admin role)
	admin := auth.Group("")
	admin.Use(customMiddleware.AdminMiddleware())

	// Theme installation endpoint (requires admin permission)
	admin.POST("/theme/install", func(c echo.Context) error {
		// Parse request
		var req ThemeInstallRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format",
			})
		}

		if req.ThemeURL == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Theme URL is required",
			})
		}

		// Create temporary file for theme config
		tmpFile, err := ioutil.TempFile("", "theme-*.json")
		if err != nil {
			logger.Log.Errorf("Failed to create temp file: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create temporary file",
			})
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		// Download theme config to temp file
		logger.Log.Infof("Downloading theme config from %s", req.ThemeURL)
		if err := downloadFile(req.ThemeURL, tmpFile.Name()); err != nil {
			logger.Log.Errorf("Failed to download theme config: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to download theme config: %v", err),
			})
		}

		// Install theme
		logger.Log.Infof("Installing theme from %s", tmpFile.Name())
		installedTheme, err := installTheme(tmpFile.Name(), webDir)
		if err != nil {
			logger.Log.Errorf("Failed to install theme: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to install theme: %v", err),
			})
		}

		// Update current theme
		themeBytes, err := ioutil.ReadFile(tmpFile.Name())
		if err != nil {
			logger.Log.Errorf("Failed to read theme config: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to read theme config",
			})
		}

		// Copy theme config to the config directory
		configDir := filepath.Dir(themeFilePath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			logger.Log.Errorf("Failed to create config directory: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to create config directory",
			})
		}

		if err := ioutil.WriteFile(themeFilePath, themeBytes, 0644); err != nil {
			logger.Log.Errorf("Failed to write theme config: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to update theme config",
			})
		}

		// Reload theme
		if _, err := theme.LoadTheme(themeFilePath); err != nil {
			logger.Log.Warnf("Failed to reload theme: %v", err)
		}

		logger.Log.Infof("Theme installed successfully: %s", installedTheme.Name)
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message":     "Theme installed successfully",
			"name":        installedTheme.Name,
			"authors":     installedTheme.Authors,
			"version":     installedTheme.Version,
			"description": installedTheme.Description,
			"directory":   installedTheme.Directory,
		})
	})

	// List all users (admin only)
	admin.GET("/users", func(c echo.Context) error {
		store := user.GetUserStore()
		if store == nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "User store not initialized",
			})
		}

		var userList []UserListItem
		for _, u := range store {
			userList = append(userList, UserListItem{
				ID:        u.ID,
				Username:  u.Username,
				Role:      u.Role,
				IsActive:  u.IsActive,
				LastLogin: u.LastLogin,
				CreatedAt: u.CreatedAt,
			})
		}

		// Sort users by ID
		sort.Slice(userList, func(i, j int) bool {
			return userList[i].ID < userList[j].ID
		})

		return c.JSON(http.StatusOK, userList)
	})

	// Create a new user (admin only)
	admin.POST("/users", func(c echo.Context) error {
		var req CreateUserRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format",
			})
		}

		// Validate request
		if req.Username == "" || req.Password == "" || req.Role == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Username, password and role are required",
			})
		}

		// Get next user ID
		nextID := user.GetNextUserID()

		// Create user
		if err := user.CreateUser(req.Username, req.Password, req.Role, nextID, req.IsActive); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to create user: %v", err),
			})
		}

		// Get the created user
		newUser, err := user.GetUser(req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "User created but could not retrieve details",
			})
		}

		return c.JSON(http.StatusCreated, UserListItem{
			ID:        newUser.ID,
			Username:  newUser.Username,
			Role:      newUser.Role,
			IsActive:  newUser.IsActive,
			CreatedAt: newUser.CreatedAt,
		})
	})

	// Update user status (admin only)
	admin.PATCH("/users/:username/status", func(c echo.Context) error {
		username := c.Param("username")
		if username == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Username is required",
			})
		}

		var req UpdateUserStatusRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request format",
			})
		}

		// Update user status
		if err := user.SetUserActive(username, req.IsActive); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to update user status: %v", err),
			})
		}

		// Get the updated user
		updatedUser, err := user.GetUser(username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Status updated but could not retrieve user details",
			})
		}

		return c.JSON(http.StatusOK, UserListItem{
			ID:        updatedUser.ID,
			Username:  updatedUser.Username,
			Role:      updatedUser.Role,
			IsActive:  updatedUser.IsActive,
			LastLogin: updatedUser.LastLogin,
			CreatedAt: updatedUser.CreatedAt,
		})
	})

	// Serve the entire web directory under the entry point
	e.Static(fmt.Sprintf("/%s", entryPoint), webDir)

	// Handle favicon.ico
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.File(filepath.Join(webDir, "favicon.ico"))
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Log.Infof("Server starting on %s", addr)
	logger.Log.Infof("Serving web directory: %s", webDir)
	if err := e.Start(addr); err != nil {
		logger.Log.Fatal(err)
	}
}
