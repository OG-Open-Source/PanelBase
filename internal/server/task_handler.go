package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/executor"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/gorilla/mux"
)

// TaskHandler manages task-related API endpoints
type TaskHandler struct {
	executor       *executor.Executor
	config         *config.Config
	authMiddleware *middleware.AuthMiddleware
}

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	Name     string             `json:"name"`
	Commands []executor.Command `json:"commands"`
	WorkDir  string             `json:"work_dir"`
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(e *executor.Executor, cfg *config.Config, authMiddleware *middleware.AuthMiddleware) *TaskHandler {
	return &TaskHandler{
		executor:       e,
		config:         cfg,
		authMiddleware: authMiddleware,
	}
}

// RegisterRoutes registers task-related API routes
func (h *TaskHandler) RegisterRoutes(r *mux.Router) {
	// All task routes require authentication
	r.Use(h.authMiddleware.Authenticate)

	r.HandleFunc("", h.CreateTask).Methods("POST")
	r.HandleFunc("", h.ListTasks).Methods("GET")
	r.HandleFunc("/{id}", h.GetTask).Methods("GET")
	r.HandleFunc("/{id}/start", h.StartTask).Methods("POST")
	r.HandleFunc("/{id}/stop", h.StopTask).Methods("POST")
}

// CreateTask creates a new task
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	// Get the current user from the request context
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to create tasks
	if !user.HasPermission(currentUser, user.PermCreateTask) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.WorkDir == "" {
		req.WorkDir = h.config.WorkDir
	}

	task, err := h.executor.CreateTask(req.Name, req.Commands, req.WorkDir)
	if err != nil {
		logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// GetTask returns a specific task by ID
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	// Get the current user from the request context
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to view tasks
	if !user.HasPermission(currentUser, user.PermViewTask) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	task, err := h.executor.GetTaskStatus(taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// StartTask starts a task
func (h *TaskHandler) StartTask(w http.ResponseWriter, r *http.Request) {
	// Get the current user from the request context
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to run tasks
	if !user.HasPermission(currentUser, user.PermRunTask) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := h.executor.StartTask(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get updated task status
	task, err := h.executor.GetTaskStatus(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return human-readable format using UTC time
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Task: %s (%s) completed\n", task.Name, task.ID)
	fmt.Fprintf(w, "Status: %s\n", task.Status)
	fmt.Fprintf(w, "Start Time: %s\n", time.Unix(task.StartTime, 0).UTC().Format(time.RFC3339))
	fmt.Fprintf(w, "End Time: %s\n", time.Unix(task.EndTime, 0).UTC().Format(time.RFC3339))
	fmt.Fprintf(w, "Exit Code: %d\n", task.ExitCode)
	if task.Error != "" {
		fmt.Fprintf(w, "Error: %s\n", task.Error)
	}
	fmt.Fprintf(w, "\nOutput:\n%s\n", task.Output)
}

// StopTask stops a running task
func (h *TaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	// Get the current user from the request context
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to stop tasks
	if !user.HasPermission(currentUser, user.PermStopTask) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := h.executor.StopTask(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ListTasks returns a list of all tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Get the current user from the request context
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user has permission to view tasks
	if !user.HasPermission(currentUser, user.PermViewTask) {
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	}

	tasks, err := h.executor.ListTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
