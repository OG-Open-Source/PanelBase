package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/OG-Open-Source/PanelBase/pkg/theme"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// ThemeHandler handles theme-related requests
type ThemeHandler struct {
	themeConfig *theme.ThemeConfig
}

// NewThemeHandler creates a new theme handler
func NewThemeHandler(themeConfig *theme.ThemeConfig) *ThemeHandler {
	return &ThemeHandler{
		themeConfig: themeConfig,
	}
}

// RegisterRoutes registers theme routes
func (h *ThemeHandler) RegisterRoutes(e *echo.Echo, entryPath string) {
	// Get the theme directory
	themeDir, err := h.themeConfig.GetThemeDirectory()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get theme directory")
	}

	// Create a new group with the entry path
	g := e.Group(fmt.Sprintf("/%s", entryPath))

	// Serve theme files
	g.Static("/", themeDir)

	// Serve index.html as the root
	g.GET("", func(c echo.Context) error {
		return c.File(filepath.Join(themeDir, "index.html"))
	})

	// Add handler for theme metadata
	g.GET("/theme/info", h.GetThemeInfo)
}

// GetThemeInfo returns information about the current theme
func (h *ThemeHandler) GetThemeInfo(c echo.Context) error {
	currentTheme, err := h.themeConfig.GetCurrentTheme()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get current theme: %v", err),
		})
	}

	// Return full theme information, including directory and structure
	return c.JSON(http.StatusOK, currentTheme)
}
