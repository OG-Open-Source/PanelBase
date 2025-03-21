package user

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// User represents a user in the system
type User struct {
	ID           int       `json:"id"` // 用戶ID，admin為0
	Username     string    `json:"username"`
	Password     string    `json:"password"`  // Stored hashed in production
	Role         string    `json:"role"`      // "admin", "user", etc.
	IsActive     bool      `json:"is_active"` // 用戶是否可用
	APIKey       string    `json:"api_key,omitempty"`
	LastLogin    time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	APIKeyExpiry time.Time `json:"api_key_expiry,omitempty"`
}

// UserStore represents the user storage
type UserStore struct {
	Users map[string]*User `json:"users"`
	mu    sync.RWMutex
}

var (
	userStorePath string
	store         *UserStore
	jwtSecret     []byte
)

// Init initializes the user store
func Init(userStoreFile string, secret string) error {
	userStorePath = userStoreFile
	jwtSecret = []byte(secret)

	store = &UserStore{
		Users: make(map[string]*User),
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(userStorePath); os.IsNotExist(err) {
		return saveStore()
	}

	// Read and parse the file
	return loadStore()
}

// loadStore loads the user store from file
func loadStore() error {
	data, err := ioutil.ReadFile(userStorePath)
	if err != nil {
		return fmt.Errorf("error reading user store: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	if err := json.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("error parsing user store: %v", err)
	}

	return nil
}

// saveStore saves the user store to file
func saveStore() error {
	store.mu.RLock()
	defer store.mu.RUnlock()

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing user store: %v", err)
	}

	if err := ioutil.WriteFile(userStorePath, data, 0644); err != nil {
		return fmt.Errorf("error writing user store: %v", err)
	}

	return nil
}

// GetUser gets a user by username
func GetUser(username string) (*User, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	user, exists := store.Users[username]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	return user, nil
}

// CreateUser creates a new user
func CreateUser(username, password, role string, id int, isActive bool) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if _, exists := store.Users[username]; exists {
		return fmt.Errorf("user already exists: %s", username)
	}

	// In a production environment, you would hash the password
	store.Users[username] = &User{
		ID:        id,
		Username:  username,
		Password:  password, // Should be hashed
		Role:      role,
		IsActive:  isActive,
		CreatedAt: time.Now(),
	}

	return saveStore()
}

// Authenticate authenticates a user
func Authenticate(username, password string) (*User, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	user, exists := store.Users[username]
	if !exists {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// In a production environment, you would compare hashed passwords
	if user.Password != password {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login time
	user.LastLogin = time.Now()

	// We'll save this outside the lock
	go saveStore() // Fire and forget, ignoring errors

	return user, nil
}

// GenerateJWT generates a JWT token for a user
func GenerateJWT(username string, expiryHours int) (string, error) {
	user, err := GetUser(username)
	if err != nil {
		return "", err
	}

	// Check if user is active
	if !user.IsActive {
		return "", fmt.Errorf("user account is disabled")
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"id":       user.ID,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * time.Duration(expiryHours)).Unix(),
	})

	// Sign and get the complete encoded token as a string
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("error signing token: %v", err)
	}

	return tokenString, nil
}

// VerifyJWT verifies a JWT token
func VerifyJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

// GenerateAPIKey generates an API key for a user
func GenerateAPIKey(username string, expiryDays int) (string, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	user, exists := store.Users[username]
	if !exists {
		return "", fmt.Errorf("user not found: %s", username)
	}

	// Check if user is active
	if !user.IsActive {
		return "", fmt.Errorf("user account is disabled")
	}

	// Generate a JWT token with longer expiry for API key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"id":       user.ID,
		"role":     user.Role,
		"type":     "api_key",
		"exp":      time.Now().Add(time.Hour * 24 * time.Duration(expiryDays)).Unix(),
	})

	// Sign and get the complete encoded token as a string
	apiKey, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("error signing token: %v", err)
	}

	// Update user's API key
	user.APIKey = apiKey
	user.APIKeyExpiry = time.Now().Add(time.Hour * 24 * time.Duration(expiryDays))

	// Save the updated user store
	if err := saveStore(); err != nil {
		return "", fmt.Errorf("error saving user store: %v", err)
	}

	return apiKey, nil
}

// VerifyAPIKey verifies an API key
func VerifyAPIKey(apiKey string) (*User, error) {
	claims, err := VerifyJWT(apiKey)
	if err != nil {
		return nil, err
	}

	// Check if it's an API key
	keyType, ok := claims["type"].(string)
	if !ok || keyType != "api_key" {
		return nil, fmt.Errorf("invalid API key type")
	}

	// Get the username from claims
	username, ok := claims["username"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid API key")
	}

	// Get the user
	user, err := GetUser(username)
	if err != nil {
		return nil, err
	}

	// Check if the API key matches
	if user.APIKey != apiKey {
		return nil, fmt.Errorf("invalid API key")
	}

	// Check if the API key has expired
	if user.APIKeyExpiry.Before(time.Now()) {
		return nil, fmt.Errorf("API key has expired")
	}

	return user, nil
}

// IsAdmin checks if a user has admin role
func IsAdmin(username string) bool {
	user, err := GetUser(username)
	if err != nil {
		return false
	}

	return user.Role == "admin"
}

// GetNextUserID gets the next available user ID
func GetNextUserID() int {
	store.mu.RLock()
	defer store.mu.RUnlock()

	maxID := 0
	for _, user := range store.Users {
		if user.ID > maxID {
			maxID = user.ID
		}
	}

	return maxID + 1
}

// SetUserActive sets the active status of a user
func SetUserActive(username string, isActive bool) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	user, exists := store.Users[username]
	if !exists {
		return fmt.Errorf("user not found: %s", username)
	}

	user.IsActive = isActive
	return saveStore()
}

// GetUserStore returns a copy of all users in the store
func GetUserStore() map[string]*User {
	store.mu.RLock()
	defer store.mu.RUnlock()

	// Create a copy of the user map to avoid concurrent access issues
	usersCopy := make(map[string]*User, len(store.Users))
	for k, v := range store.Users {
		// Create a deep copy of each user
		userCopy := *v
		usersCopy[k] = &userCopy
	}

	return usersCopy
}
