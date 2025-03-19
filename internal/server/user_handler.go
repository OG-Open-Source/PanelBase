package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// UserHandler manages user-related API endpoints
type UserHandler struct {
	userManager    user.UserManager
	authMiddleware *middleware.AuthMiddleware
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userManager user.UserManager, authMiddleware *middleware.AuthMiddleware) *UserHandler {
	return &UserHandler{
		userManager:    userManager,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers user-related API routes
func (h *UserHandler) RegisterRoutes(r *mux.Router) {
	// Public routes - no auth required
	r.HandleFunc("/auth/login", h.Login).Methods("POST")

	// Protected routes - require authentication
	secured := r.PathPrefix("/users").Subrouter()
	secured.Use(h.authMiddleware.Authenticate)

	secured.HandleFunc("", h.ListUsers).Methods("GET")
	secured.HandleFunc("", h.CreateUser).Methods("POST")
	secured.HandleFunc("/{id:[0-9]+}", h.GetUser).Methods("GET")
	secured.HandleFunc("/{id:[0-9]+}", h.UpdateUser).Methods("PUT")
	secured.HandleFunc("/{id:[0-9]+}", h.DeleteUser).Methods("DELETE")
	secured.HandleFunc("/{id:[0-9]+}/api-key", h.GenerateAPIKey).Methods("POST")
}

// Login handles user login and returns a JWT token
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	authenticatedUser, err := h.userManager.Authenticate(credentials.Username, credentials.Password)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed login attempt for username: %s", credentials.Username))
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := h.authMiddleware.GenerateToken(authenticatedUser)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to generate token: %v", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"user":  authenticatedUser,
		"token": token,
	}

	logger.Info(fmt.Sprintf("User logged in: %s (ID: %d)", authenticatedUser.Username, authenticatedUser.ID))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListUsers returns a list of all users
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Get the current user from the request context
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to view users
	if !user.HasPermission(currentUser, user.PermViewUser) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	users, err := h.userManager.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// GetUser returns a specific user by ID
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to view users
	if !user.HasPermission(currentUser, user.PermViewUser) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Get user ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get user by ID
	requestedUser, err := h.userManager.GetUser(id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requestedUser)
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to create users
	if !user.HasPermission(currentUser, user.PermCreateUser) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	var newUser struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	// 读取请求体内容并记录日志
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to read request body: %v", err))
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// 记录请求数据（不包含密码）
	logger.Debug(fmt.Sprintf("CreateUser request body: %s", string(bodyBytes)))

	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		logger.Error(fmt.Sprintf("Failed to decode JSON: %v", err))
		http.Error(w, "Invalid request format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 记录解析后的信息（不包含密码）
	logger.Debug(fmt.Sprintf("Creating user - Username: %s, Role: %s", newUser.Username, newUser.Role))

	// Validate inputs
	if newUser.Username == "" || newUser.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// 将字符串转换为UserRole类型
	userRole := user.UserRole(newUser.Role)

	// 确保角色有效
	validRoles := map[user.UserRole]bool{
		user.RoleRoot:  true,
		user.RoleAdmin: true,
		user.RoleUser:  true,
		user.RoleGuest: true,
	}

	if !validRoles[userRole] {
		logger.Error(fmt.Sprintf("Invalid role provided: %s", newUser.Role))
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Root users can only be created by other root users
	if userRole == user.RoleRoot && currentUser.Role != user.RoleRoot {
		http.Error(w, "Only ROOT users can create other ROOT users", http.StatusForbidden)
		return
	}

	createdUser, err := h.userManager.CreateUser(newUser.Username, newUser.Password, userRole)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create user: %v", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.Info(fmt.Sprintf("User created successfully: %s with role %s", newUser.Username, userRole))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdUser)
}

// UpdateUser updates an existing user
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to update users
	if !user.HasPermission(currentUser, user.PermUpdateUser) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Get user ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get existing user
	existingUser, err := h.userManager.GetUser(id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Only root users can update other root users
	if existingUser.Role == user.RoleRoot && currentUser.Role != user.RoleRoot {
		http.Error(w, "Only ROOT users can update ROOT users", http.StatusForbidden)
		return
	}

	// Parse update request
	var updateRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Apply updates
	if err := h.userManager.UpdateUser(id, updateRequest); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated user
	updatedUser, err := h.userManager.GetUser(id)
	if err != nil {
		http.Error(w, "Failed to retrieve updated user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to delete users
	if !user.HasPermission(currentUser, user.PermDeleteUser) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Get user ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Cannot delete yourself
	if id == currentUser.ID {
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	// Get existing user to check role
	existingUser, err := h.userManager.GetUser(id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Only root users can delete other root users
	if existingUser.Role == user.RoleRoot && currentUser.Role != user.RoleRoot {
		http.Error(w, "Only ROOT users can delete ROOT users", http.StatusForbidden)
		return
	}

	// Delete the user
	if err := h.userManager.DeleteUser(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GenerateAPIKey generates a new API key for a user
func (h *UserHandler) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to update users
	if !user.HasPermission(currentUser, user.PermUpdateUser) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	// Get user ID from URL
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get existing user
	existingUser, err := h.userManager.GetUser(id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Only root users can update API keys for other root users
	if existingUser.Role == user.RoleRoot && currentUser.Role != user.RoleRoot && currentUser.ID != id {
		http.Error(w, "Only ROOT users can update API keys for ROOT users", http.StatusForbidden)
		return
	}

	// Generate a new API key (UUID)
	apiKey := uuid.New().String()
	logger.Debug(fmt.Sprintf("Generated API key: %s for user: %s (ID: %d)", apiKey, existingUser.Username, id))

	// Update the user with the new API key
	updates := map[string]interface{}{
		"api_key": apiKey,
	}

	logger.Debug(fmt.Sprintf("Updating user with new API key. Updates: %+v", updates))
	if err := h.userManager.UpdateUser(id, updates); err != nil {
		logger.Error(fmt.Sprintf("Failed to update user with new API key: %v", err))
		http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
		return
	}

	// Verify the update was successful
	updatedUser, err := h.userManager.GetUser(id)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to retrieve updated user: %v", err))
	} else {
		logger.Debug(fmt.Sprintf("API Key verification - User: %s, Has API Key: %t",
			updatedUser.Username, updatedUser.APIKey != ""))
	}

	// Return the new API key
	response := map[string]string{
		"api_key": apiKey,
	}

	logger.Info(fmt.Sprintf("Generated new API key for user %s (ID: %d)", existingUser.Username, id))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
