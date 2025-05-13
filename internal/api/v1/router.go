package v1

import (
	"fmt"
	"net/http"
)

// NewRouter creates a new router for the v1 API endpoints.
func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Example endpoint
	mux.HandleFunc("/status", handleStatus) // Path will be /api/v1/status

	// SSE endpoint
	mux.HandleFunc("/events", handleSSE) // Path will be /api/v1/events
	// TODO: Add other v1 API endpoints

	return mux
}

// handleStatus is a placeholder handler for the /status endpoint.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, `{"status": "API v1 running"}`)
}
