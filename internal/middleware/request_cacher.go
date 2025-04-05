package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const ContextKeyRequestBody = "requestBodyBytes"

// CacheRequestBody reads the request body, stores it in the context,
// and replaces the original body reader so it can be read again later.
// Note: This should be placed before any middleware or handler that needs the request body.
func CacheRequestBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only cache for methods that typically have bodies and are relevant for logging
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch || c.Request.Method == http.MethodDelete {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				// Handle error reading body, maybe log it and continue without caching?
				// Or abort? Let's log and continue for now.
				// log.Printf("Warning: Error reading request body for caching: %v", err)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(nil)) // Set empty body
			} else {
				// Store the read bytes in the context
				c.Set(ContextKeyRequestBody, bodyBytes)
				// Replace the request body with a new reader based on the read bytes
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		} else {
			// For other methods (GET, etc.), set an empty byte slice or nil
			c.Set(ContextKeyRequestBody, []byte{})
		}
		c.Next()
	}
}
 