package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response defines the standard API response structure.
type Response struct {
	Status  string      `json:"status"`         // "success" or "error"
	Message string      `json:"message"`        // Descriptive message
	Data    interface{} `json:"data,omitempty"` // Response data (omitted if empty or null for errors)
}

// SuccessResponse sends a standardized success JSON response (HTTP 200 OK).
func SuccessResponse(c *gin.Context, message string, data interface{}) {
	response := Response{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	// If data is nil, the omitempty tag will exclude it
	c.JSON(http.StatusOK, response)
}

// ErrorResponse sends a standardized error JSON response with the given HTTP status code.
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	response := Response{
		Status:  "error",
		Message: message,
		Data:    nil, // Explicitly set data to nil for errors
	}
	c.JSON(statusCode, response)
	// Note: We still use the statusCode for the HTTP header, but the body has the standard format.
}

// --- Example Usage in a Handler ---
/*
func ExampleHandler(c *gin.Context) {
	// Simulate getting data
	data, err := someService.GetData()
	if err != nil {
		// Log the detailed error internally
		log.Printf("Error getting data: %v", err)
		// Send generic error response to client
		ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve data.")
		return
	}

	// Simulate validation error
	if someValue := c.Query("value"); someValue == "" {
		ErrorResponse(c, http.StatusBadRequest, "Missing required query parameter 'value'.")
		return
	}

	// Send success response
	SuccessResponse(c, "Data retrieved successfully.", data)
}
*/
