package ui_settings

import (
	"net/http"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/gin-gonic/gin"
)

// GetSettingsHandler retrieves the current UI settings.
// GET /api/v1/settings/ui
func GetSettingsHandler(c *gin.Context) {
	// Permission check is handled by middleware RequirePermission("settings", "read")

	settings, err := GetUISettings()
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve UI settings: "+err.Error())
		return
	}
	server.SuccessResponse(c, "UI settings retrieved successfully", settings)
}

// UpdateSettingsHandler handles updating UI settings.
// PUT /api/v1/settings/ui
func UpdateSettingsHandler(c *gin.Context) {
	// Permission check is handled by middleware RequirePermission("settings", "update")

	// Bind request body to a map for partial updates
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if len(updateData) == 0 {
		server.ErrorResponse(c, http.StatusBadRequest, "Request payload cannot be empty for update")
		return
	}

	// Call the service function with the map
	updatedSettings, err := UpdateUISettings(updateData)
	if err != nil {
		// Handle potential errors like "not loaded" or save failure
		if strings.Contains(err.Error(), "not loaded") {
			server.ErrorResponse(c, http.StatusInternalServerError, "UI settings not loaded on server")
		} else if strings.Contains(err.Error(), "failed to save") {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to save updated UI settings")
		} else {
			server.ErrorResponse(c, http.StatusInternalServerError, "Error updating UI settings: "+err.Error())
		}
		return
	}

	server.SuccessResponse(c, "UI settings updated successfully", updatedSettings)
}
