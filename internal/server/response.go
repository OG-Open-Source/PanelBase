package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response defines the standard API response structure.
type Response struct {
	Status  string      `json:"status"`          // "success" or "error" (using "error" instead of "failure" for common practice)
	Message string      `json:"message"`         // User-friendly message describing the result
	Data    interface{} `json:"data,omitempty"`  // Optional data payload (omitted if nil)
	Error   string      `json:"error,omitempty"` // Optional error details (e.g., validation errors, internal error code)
}

// SuccessResponse sends a standardized success JSON response.
// Use this for successful operations (2xx status codes).
func SuccessResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// ErrorResponse sends a standardized error JSON response.
// Use this for client or server errors (4xx, 5xx status codes).
// The 'errInfo' parameter can provide additional context about the error (e.g., validation details, internal error code).
func ErrorResponse(c *gin.Context, statusCode int, message string, errInfo ...string) {
	var errorDetail string
	if len(errInfo) > 0 {
		errorDetail = errInfo[0]
	}

	// Avoid sending detailed internal errors in production unless specifically needed.
	// Consider logging the full error internally.
	if statusCode >= 500 && gin.Mode() == gin.ReleaseMode {
		message = "An internal server error occurred." // Generic message for 5xx in release mode
		errorDetail = ""                               // Don't expose internal details
	}

	c.JSON(statusCode, Response{
		Status:  "error", // Use "error" consistently
		Message: message,
		Error:   errorDetail,
	})
	c.Abort() // Optional: Abort further middleware execution after sending an error
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
		ErrorResponse(c, http.StatusBadRequest, "Missing required query parameter 'value'.", "validation_error")
		return
	}

	// Send success response
	SuccessResponse(c, "Data retrieved successfully.", data)
}
*/
