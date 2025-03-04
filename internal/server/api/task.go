package api

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/OG-Open-Source/PanelBase/internal/executor"
)

type TaskHandler struct {
	executor *executor.Executor
}

func NewTaskHandler(e *executor.Executor) *TaskHandler {
	return &TaskHandler{executor: e}
}

func (h *TaskHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/tasks", h.CreateTask).Methods("POST")
	r.HandleFunc("/tasks/{id}", h.GetTask).Methods("GET")
	r.HandleFunc("/tasks/{id}/start", h.StartTask).Methods("POST")
	r.HandleFunc("/tasks/{id}/stop", h.StopTask).Methods("POST")
	r.HandleFunc("/tasks", h.ListTasks).Methods("GET")
}

// API 處理方法... 