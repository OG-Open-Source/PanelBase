package uisettings

import (
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/models"
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

// UpdateSettingsHandler updates the UI settings.
// PUT /api/v1/settings/ui
func UpdateSettingsHandler(c *gin.Context) {
	// Permission check is handled by middleware RequirePermission("settings", "update")

	var updates models.UISettings
	if err := c.ShouldBindJSON(&updates); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	updatedSettings, err := UpdateUISettings(updates)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to update UI settings: "+err.Error())
		return
	}

	server.SuccessResponse(c, "UI settings updated successfully", updatedSettings)
}
