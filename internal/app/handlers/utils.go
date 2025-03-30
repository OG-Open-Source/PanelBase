package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SendSuccessResponse 發送成功回應
func SendSuccessResponse(c *gin.Context, message string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

// SendErrorResponse 發送錯誤回應
func SendErrorResponse(c *gin.Context, statusCode int, message string, errorDetails map[string]interface{}) {
	if errorDetails == nil {
		errorDetails = make(map[string]interface{})
	}
	c.JSON(statusCode, gin.H{
		"status":  "failure",
		"message": message,
		"data":    errorDetails,
	})
}
