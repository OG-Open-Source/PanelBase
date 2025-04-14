package response

// ApiResponse defines the standard JSON response structure for the API.
type ApiResponse struct {
	Status  string      `json:"status"`         // "success" or "failure"
	Message string      `json:"message"`        // User-friendly message describing the result
	Data    interface{} `json:"data,omitempty"` // Optional data payload (could be object, array, or nil)
}

// Success creates a standard success response.
func Success(message string, data interface{}) ApiResponse {
	return ApiResponse{Status: "success", Message: message, Data: data}
}

// Failure creates a standard failure response (client-side errors, validation, etc.).
// Data can be nil or contain details like validation errors.
func Failure(message string, data interface{}) ApiResponse {
	return ApiResponse{Status: "failure", Message: message, Data: data}
}

// Note: We are combining 5xx server errors into the 'failure' status for simplicity,
// but distinguishing them with specific messages and appropriate HTTP status codes.
// Alternatively, an "error" status could be used for 5xx.
