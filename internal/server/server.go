package server

import (
	"log"
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/api/v1" // Import v1 handlers
)

// StartServer initializes and starts the HTTP server.
func StartServer(addr string) {
	log.Printf("Starting server on %s\n", addr)

	mux := http.NewServeMux()

	// API v1 routes
	apiV1Handler := http.StripPrefix("/api/v1", v1.NewRouter())
	mux.Handle("/api/v1/", apiV1Handler)

	// TODO: Add other routes (e.g., frontend static files)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	err := server.ListenAndServe()
	if err != nil {
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		} else if err == http.ErrServerClosed {
			log.Println("Server stopped gracefully")
		}
	}
}
