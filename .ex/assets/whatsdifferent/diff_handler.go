package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// Settings for the plugin
type Settings struct {
	IgnoreWhitespace bool `json:"ignore_whitespace"`
	ContextLines     int  `json:"context_lines"`
	MaxHistoryItems  int  `json:"max_history_items"`
}

// DiffResult represents the result of a diff operation
type DiffResult struct {
	Original     string    `json:"original"`
	New          string    `json:"new"`
	Diff         string    `json:"diff"`
	ChangesCount int       `json:"changes_count"`
	Timestamp    time.Time `json:"timestamp"`
	ID           string    `json:"id"`
}

// HistoryItem represents an item in the history
type HistoryItem struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Summary   string    `json:"summary"`
}

var (
	settings     Settings
	diffHistory  []DiffResult
	startTime    time.Time
	settingsFile = "settings.json"
	historyFile  = "history.json"
)

func init() {
	// Initialize default settings
	settings = Settings{
		IgnoreWhitespace: true,
		ContextLines:     3,
		MaxHistoryItems:  50,
	}

	// Load settings from file if it exists
	if _, err := os.Stat(settingsFile); err == nil {
		data, err := os.ReadFile(settingsFile)
		if err == nil {
			json.Unmarshal(data, &settings)
		}
	}

	// Load history from file if it exists
	if _, err := os.Stat(historyFile); err == nil {
		data, err := os.ReadFile(historyFile)
		if err == nil {
			json.Unmarshal(data, &diffHistory)
		}
	}

	startTime = time.Now()
}

// HandleDiff processes diff requests
func HandleDiff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// Return sample diff data for GET requests
		sampleData := map[string]interface{}{
			"status":  "success",
			"message": "Sample diff data. Use POST to perform actual diff operation.",
			"sample": DiffResult{
				Original:     "This is the original text.",
				New:          "This is the new modified text.",
				Diff:         "Differences would appear here.",
				ChangesCount: 1,
				Timestamp:    time.Now(),
				ID:           "sample-id",
			},
		}
		json.NewEncoder(w).Encode(sampleData)

	case "POST":
		// Process actual diff request
		var requestData struct {
			Original string `json:"original"`
			New      string `json:"new"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestData); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Perform diff operation
		result := performDiff(requestData.Original, requestData.New)

		// Save to history
		saveToHistory(result)

		// Return result
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   result,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleStatus returns the plugin status
func HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	uptime := time.Since(startTime).String()
	status := map[string]interface{}{
		"status":  "running",
		"version": "1.0.0",
		"uptime":  uptime,
	}

	json.NewEncoder(w).Encode(status)
}

// HandleHistory manages diff history
func HandleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// Get history items
		limit := 10
		if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
			fmt.Sscanf(limitParam, "%d", &limit)
			if limit <= 0 {
				limit = 10
			}
		}

		id := r.URL.Query().Get("id")
		if id != "" {
			// Return specific history item
			for _, item := range diffHistory {
				if item.ID == id {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status": "success",
						"data":   item,
					})
					return
				}
			}
			http.Error(w, "History item not found", http.StatusNotFound)
			return
		}

		// Return all history items (limited)
		historyLimit := len(diffHistory)
		if historyLimit > limit {
			historyLimit = limit
		}

		summaries := make([]HistoryItem, 0, historyLimit)
		for i := 0; i < historyLimit; i++ {
			idx := len(diffHistory) - 1 - i
			if idx < 0 {
				break
			}
			item := diffHistory[idx]
			summaries = append(summaries, HistoryItem{
				ID:        item.ID,
				Timestamp: item.Timestamp,
				Summary:   createSummary(item),
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"items":  summaries,
			"count":  len(diffHistory),
		})

	case "POST":
		// Create a new history entry (perform diff and save)
		var requestData struct {
			Original string `json:"original"`
			New      string `json:"new"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestData); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Perform diff operation
		result := performDiff(requestData.Original, requestData.New)

		// Save to history
		saveToHistory(result)

		// Return result
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   result,
		})

	case "DELETE":
		// Delete history
		var requestData struct {
			ID string `json:"id"`
		}

		// If ID is provided, delete specific item
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requestData); err == nil && requestData.ID != "" {
			newHistory := make([]DiffResult, 0, len(diffHistory))
			for _, item := range diffHistory {
				if item.ID != requestData.ID {
					newHistory = append(newHistory, item)
				}
			}

			if len(newHistory) == len(diffHistory) {
				http.Error(w, "History item not found", http.StatusNotFound)
				return
			}

			diffHistory = newHistory
			saveHistoryToFile()

			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "History item deleted",
			})
			return
		}

		// Otherwise, clear entire history
		diffHistory = []DiffResult{}
		saveHistoryToFile()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "History cleared",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleSettings manages plugin settings
func HandleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// Return current settings
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"settings": settings,
		})

	case "PUT", "PATCH":
		// Update settings
		var newSettings Settings

		if r.Method == "PUT" {
			// Replace all settings
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&newSettings); err != nil {
				http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
				return
			}
			settings = newSettings
		} else {
			// Patch existing settings
			var patchData map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&patchData); err != nil {
				http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
				return
			}

			// Apply patches
			if val, ok := patchData["ignore_whitespace"]; ok {
				if boolVal, ok := val.(bool); ok {
					settings.IgnoreWhitespace = boolVal
				}
			}
			if val, ok := patchData["context_lines"]; ok {
				if floatVal, ok := val.(float64); ok {
					settings.ContextLines = int(floatVal)
				}
			}
			if val, ok := patchData["max_history_items"]; ok {
				if floatVal, ok := val.(float64); ok {
					settings.MaxHistoryItems = int(floatVal)
				}
			}
		}

		// Validate settings
		if settings.ContextLines < 0 {
			settings.ContextLines = 0
		} else if settings.ContextLines > 10 {
			settings.ContextLines = 10
		}

		if settings.MaxHistoryItems < 1 {
			settings.MaxHistoryItems = 1
		} else if settings.MaxHistoryItems > 100 {
			settings.MaxHistoryItems = 100
		}

		// Save settings
		saveSettingsToFile()

		// Return updated settings
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"message":  "Settings updated",
			"settings": settings,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper Functions

// performDiff performs the actual diff operation
func performDiff(original, new string) DiffResult {
	dmp := diffmatchpatch.New()

	// Preprocess text based on settings
	if settings.IgnoreWhitespace {
		original = strings.TrimSpace(original)
		new = strings.TrimSpace(new)
	}

	// Perform diff
	diffs := dmp.DiffMain(original, new, false)
	if settings.ContextLines > 0 {
		diffs = dmp.DiffCleanupSemantic(diffs)
	}

	// Generate pretty HTML diff
	diffHTML := dmp.DiffPrettyHtml(diffs)

	// Count changes
	changesCount := 0
	for _, diff := range diffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			changesCount++
		}
	}

	// Create result
	result := DiffResult{
		Original:     original,
		New:          new,
		Diff:         diffHTML,
		ChangesCount: changesCount,
		Timestamp:    time.Now(),
		ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	return result
}

// saveToHistory adds a diff result to history
func saveToHistory(result DiffResult) {
	// Add to history
	diffHistory = append(diffHistory, result)

	// Limit history size
	if len(diffHistory) > settings.MaxHistoryItems {
		diffHistory = diffHistory[len(diffHistory)-settings.MaxHistoryItems:]
	}

	// Save to file
	saveHistoryToFile()
}

// createSummary creates a short summary of a diff result
func createSummary(result DiffResult) string {
	const maxLen = 50

	originalShort := result.Original
	if len(originalShort) > maxLen {
		originalShort = originalShort[:maxLen] + "..."
	}

	return fmt.Sprintf("%d changes, original: %s", result.ChangesCount, originalShort)
}

// saveSettingsToFile saves settings to file
func saveSettingsToFile() {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(settingsFile, data, 0644)
}

// saveHistoryToFile saves history to file
func saveHistoryToFile() {
	data, err := json.Marshal(diffHistory)
	if err != nil {
		return
	}
	os.WriteFile(historyFile, data, 0644)
}

// Main handler for registering HTTP endpoints
func main() {
	http.HandleFunc("/diff", HandleDiff)
	http.HandleFunc("/status", HandleStatus)
	http.HandleFunc("/history", HandleHistory)
	http.HandleFunc("/settings", HandleSettings)

	// Server is started by the PanelBase plugin system
}
