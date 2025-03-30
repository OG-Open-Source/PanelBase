package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads the application configuration from a YAML file
func LoadConfig(configPath string) (*models.Config, error) {
	configFile, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer configFile.Close()

	configData, err := io.ReadAll(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadUsersConfig loads the users configuration from a JSON file
func LoadUsersConfig(filePath string) (*models.UsersConfig, error) {
	usersFile, err := os.Open(filePath)

	// 如果文件不存在，创建默认配置
	if os.IsNotExist(err) {
		log.Printf("User config file %s not found, creating default configuration\n", filePath)

		// 生成隨機用戶ID
		userID := generateRandomSecret(16)

		// 创建默认配置
		config := &models.UsersConfig{
			Users: map[string]*models.User{
				userID: {
					// ID 欄位已被移除
					IsActive:  true,
					Username:  "admin",
					Password:  "$2a$10$VwPWUIUaqO6O7Ti1DZpHPeJKdrK4sUuutuBcLUpkeeZHiGuswRQey", // bcrypt hashed "admin"
					Role:      "admin",
					Name:      "Administrator",
					Email:     "admin@example.com",
					CreatedAt: models.JsonTime(time.Now()),
					LastLogin: models.JsonTime(time.Time{}), // 使用零值表示從未登入
					API:       make(map[string]models.APIToken),
				},
			},
			DefaultRole: "user",
			JWTSecret:   generateRandomSecret(32),
		}

		// 确保目录存在
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		// 保存配置
		configData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to serialize default user config: %w", err)
		}

		if err := os.WriteFile(filePath, configData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save default user config: %w", err)
		}

		return config, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to open user config file: %w", err)
	}
	defer usersFile.Close()

	usersData, err := io.ReadAll(usersFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config file: %w", err)
	}

	var usersConfig models.UsersConfig
	if err := json.Unmarshal(usersData, &usersConfig); err != nil {
		return nil, fmt.Errorf("failed to parse user config file: %w", err)
	}

	// 确保Users不为nil
	if usersConfig.Users == nil {
		usersConfig.Users = make(map[string]*models.User)
	}

	// 如果JWTSecret为空，生成一个
	if usersConfig.JWTSecret == "" {
		usersConfig.JWTSecret = generateRandomSecret(32)

		// 保存更新后的配置
		updatedData, err := json.MarshalIndent(usersConfig, "", "  ")
		if err == nil {
			_ = os.WriteFile(filePath, updatedData, 0644)
		}
	}

	return &usersConfig, nil
}

// 生成随机密钥
func generateRandomSecret(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// LoadThemesConfig loads the themes configuration from a JSON file
func LoadThemesConfig(filePath string) (*models.ThemesConfig, error) {
	themesFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open themes file: %w", err)
	}
	defer themesFile.Close()

	themesData, err := io.ReadAll(themesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read themes file: %w", err)
	}

	var themesConfig models.ThemesConfig
	if err := json.Unmarshal(themesData, &themesConfig); err != nil {
		return nil, fmt.Errorf("failed to parse themes file: %w", err)
	}

	return &themesConfig, nil
}

// LoadCommandsConfig loads the commands configuration from a JSON or YAML file
func LoadCommandsConfig(filePath string) (*models.SystemCommandsConfig, error) {
	if filePath == "" {
		// 如果未提供路径，使用默认的空配置
		return &models.SystemCommandsConfig{Enabled: false, Routes: make(map[string]models.CommandConfig)}, nil
	}

	// 检测文件格式
	format := "yaml" // 默认为YAML

	// 如果路径包含commands.json，确保使用JSON解析
	if strings.Contains(filePath, "commands.json") {
		format = "json"
	}

	// 打开文件
	commandsFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open commands file: %w", err)
	}
	defer commandsFile.Close()

	// 读取文件内容
	commandsData, err := io.ReadAll(commandsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read commands file: %w", err)
	}

	// 解析配置
	var commandsConfig models.SystemCommandsConfig
	if format == "yaml" {
		if err := yaml.Unmarshal(commandsData, &commandsConfig); err != nil {
			return nil, fmt.Errorf("failed to parse YAML commands file: %w", err)
		}
	} else {
		if err := json.Unmarshal(commandsData, &commandsConfig); err != nil {
			return nil, fmt.Errorf("failed to parse JSON commands file: %w", err)
		}
	}

	// 记录加载的配置数量
	log.Printf("Loaded %d command configurations\n", len(commandsConfig.Routes))

	return &commandsConfig, nil
}

// LoadPluginsConfig loads the plugins configuration
func LoadPluginsConfig(pluginsFilePath string) (*models.PluginsConfigJSON, error) {
	log.Printf("Loading plugins configuration from: %s", pluginsFilePath)

	// Check if file exists
	if _, err := os.Stat(pluginsFilePath); os.IsNotExist(err) {
		// Create default config if not exists
		log.Printf("Plugins configuration file not found, creating default configuration")
		defaultConfig := &models.PluginsConfigJSON{
			Plugins: make(map[string]models.PluginInfo),
		}

		// Save default config
		data, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal default plugins configuration: %w", err)
		}

		if err := os.WriteFile(pluginsFilePath, data, 0644); err != nil {
			return nil, fmt.Errorf("failed to write default plugins configuration: %w", err)
		}

		return defaultConfig, nil
	}

	// Read file
	data, err := os.ReadFile(pluginsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins configuration: %w", err)
	}

	// Parse JSON
	var config models.PluginsConfigJSON
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse plugins configuration: %w", err)
	}

	// Validate methods for all plugins
	for pluginName, plugin := range config.Plugins {
		for i, endpoint := range plugin.Endpoints {
			if !endpoint.ValidateMethods() {
				log.Printf("Warning: Plugin %s endpoint %s contains invalid HTTP methods. Only GET, POST, PUT, PATCH, DELETE are allowed.",
					pluginName, endpoint.Path)

				// Filter out invalid methods
				validMethods := []string{}
				for _, method := range endpoint.Methods {
					isValid := false
					for _, validMethod := range models.ValidMethods {
						if method == validMethod {
							isValid = true
							break
						}
					}
					if isValid {
						validMethods = append(validMethods, method)
					}
				}

				// If no valid methods remain, set default to GET
				if len(validMethods) == 0 {
					validMethods = []string{"GET"}
					log.Printf("No valid methods for plugin %s endpoint %s, defaulting to GET",
						pluginName, endpoint.Path)
				}

				// Update methods
				plugin.Endpoints[i].Methods = validMethods
			}
		}
		// Update plugin in map after validation
		config.Plugins[pluginName] = plugin
	}

	log.Printf("Loaded plugins configuration with %d plugins", len(config.Plugins))

	// Print loaded plugins information
	for name, plugin := range config.Plugins {
		log.Printf("Plugin: %s, Name: %s, API Version: %s, Endpoints: %d",
			name, plugin.Name, plugin.APIVersion, len(plugin.Endpoints))
	}

	return &config, nil
}

// SavePluginsConfig saves the plugins configuration to JSON file
func SavePluginsConfig(pluginsFilePath string, config *models.PluginsConfigJSON) error {
	// Validate methods before saving
	for pluginName, plugin := range config.Plugins {
		for i, endpoint := range plugin.Endpoints {
			if !endpoint.ValidateMethods() {
				// Filter out invalid methods
				validMethods := []string{}
				for _, method := range endpoint.Methods {
					isValid := false
					for _, validMethod := range models.ValidMethods {
						if method == validMethod {
							isValid = true
							break
						}
					}
					if isValid {
						validMethods = append(validMethods, method)
					}
				}

				// If no valid methods remain, set default to GET
				if len(validMethods) == 0 {
					validMethods = []string{"GET"}
				}

				// Update methods
				plugin.Endpoints[i].Methods = validMethods
			}
		}
		// Update plugin in map after validation
		config.Plugins[pluginName] = plugin
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plugins configuration: %w", err)
	}

	if err := os.WriteFile(pluginsFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plugins configuration: %w", err)
	}

	return nil
}

// SaveThemesConfig saves the themes configuration to JSON file
func SaveThemesConfig(themesFilePath string, config *models.ThemesConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal themes configuration: %w", err)
	}

	if err := os.WriteFile(themesFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write themes configuration: %w", err)
	}

	return nil
}

// SaveUsersConfig saves the users configuration to a file
func SaveUsersConfig(usersConfig *models.UsersConfig, baseDir string) error {
	configPath := filepath.Join(baseDir, "configs", "users.json")

	// Serialize the configuration
	data, err := json.MarshalIndent(usersConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize users configuration: %w", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save users configuration: %w", err)
	}

	return nil
}
