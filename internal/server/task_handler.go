package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"github.com/gorilla/mux"
	"github.com/OG-Open-Source/PanelBase/internal/executor"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/config"
)

type TaskHandler struct {
	executor *executor.Executor
	config   *config.Config
}

// 請求結構體
type CreateTaskRequest struct {
	Name     string            `json:"name"`
	Commands []executor.Command `json:"commands"`
	WorkDir  string           `json:"work_dir"`
}

func NewTaskHandler(e *executor.Executor, cfg *config.Config) *TaskHandler {
	return &TaskHandler{
		executor: e,
		config:   cfg,
	}
}

func (h *TaskHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/tasks", h.CreateTask).Methods("POST")
	r.HandleFunc("/tasks/{id}", h.GetTask).Methods("GET")
	r.HandleFunc("/tasks/{id}/start", h.StartTask).Methods("POST")
	r.HandleFunc("/tasks/{id}/stop", h.StopTask).Methods("POST")
	r.HandleFunc("/tasks", h.ListTasks).Methods("GET")
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
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

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
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

func (h *TaskHandler) StartTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := h.executor.StartTask(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 獲取更新後的任務狀態
	task, err := h.executor.GetTaskStatus(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回人類可讀的格式，使用 UTC 時間
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

func (h *TaskHandler) StopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["id"]

	if err := h.executor.StopTask(taskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.executor.ListTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
} 