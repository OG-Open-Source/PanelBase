package settings

import (
	"net/http"
	"strings"

	"github.com/OG-Open-Source/PanelBase/pkg/serverutils"
	// Import the original service package to call its functions
	"github.com/OG-Open-Source/PanelBase/pkg/uisettings"
	"github.com/gin-gonic/gin"
)

// GetSettingsHandler retrieves the current UI settings.
// GET /api/v1/settings/ui
func GetSettingsHandler(c *gin.Context) {
	// Permission check is handled by middleware RequirePermission("settings", "read")

	// Call service function from original package
	settings, err := uisettings.GetUISettings()
	if err != nil {
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve UI settings: "+err.Error())
		return
	}
	serverutils.SuccessResponse(c, "UI settings retrieved successfully", settings)
}

// UpdateSettingsHandler handles updating UI settings.
// PATCH /api/v1/settings/ui (Note: Method changed from PUT to PATCH in routes later)
func UpdateSettingsHandler(c *gin.Context) {
	// Permission check is handled by middleware RequirePermission("settings", "update")

	// Bind request body to a map for partial updates
	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if len(updateData) == 0 {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Request payload cannot be empty for update")
		return
	}

	// Call the service function from original package
	updatedSettings, err := uisettings.UpdateUISettings(updateData)
	if err != nil {
		// Handle potential errors like "not loaded" or save failure
		if strings.Contains(err.Error(), "not loaded") {
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "UI settings not loaded on server")
		} else if strings.Contains(err.Error(), "failed to save") {
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save updated UI settings")
		} else {
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Error updating UI settings: "+err.Error())
		}
		return
	}

	serverutils.SuccessResponse(c, "UI settings updated successfully", updatedSettings)
}
