package utils

import (
	"github.com/OG-Open-Source/PanelBase/internal/shared/models"
)

// FindUserIDByUsername 根據用戶名查找用戶ID（map key）
func FindUserIDByUsername(users map[string]*models.User, username string) string {
	for id, user := range users {
		if user.Username == username {
			return id
		}
	}
	return ""
}

// GetUserByUsername 根據用戶名獲取用戶和其ID
func GetUserByUsername(users map[string]*models.User, username string) (string, *models.User) {
	for id, user := range users {
		if user.Username == username {
			return id, user
		}
	}
	return "", nil
}
