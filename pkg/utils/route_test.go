package utils_test

import (
	"testing"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
)

func TestNewRouteManager(t *testing.T) {
	rm := utils.NewRouteManager()
	if rm == nil {
		t.Fatal("RouteManager initialization failed")
	}
} 