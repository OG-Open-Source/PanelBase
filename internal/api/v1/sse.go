package v1

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// handleSSE handles Server-Sent Events requests.
func handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust CORS as needed

	log.Println("SSE client connected")

	// Example: Send a message every few seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			log.Println("SSE client disconnected")
			return
		case t := <-ticker.C:
			// TODO: Replace with actual data to send
			message := fmt.Sprintf("data: {\"time\": \"%s\"}\n\n", t.Format(time.RFC3339))
			_, err := fmt.Fprint(w, message)
			if err != nil {
				log.Printf("Error writing to SSE client: %v", err)
				return // Client likely disconnected
			}
			flusher.Flush() // Flush the data to the client
			log.Println("Sent SSE message")
		}
	}
}
