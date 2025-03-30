package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"gopkg.in/yaml.v3"
)

const (
	// 配置文件相關常量
	ConfigDir      = "configs"
	CommandsFile   = "commands.json"
	ConfigYamlFile = "config.yaml"
	PluginsFile    = "plugins.json"
	ThemesFile     = "themes.json"
	UsersFile      = "users.json"

	// 基本目錄相關常量
	CommandsDir = "commands"
	PluginsDir  = "plugins"
	WebDir      = "web"

	// 端口範圍 (使用註冊端口範圍)
	MinPort = 1024  // 系統端口結束
	MaxPort = 49151 // 註冊端口結束
)

// ConfigInitializer 配置初始化器
type ConfigInitializer struct {
	BaseDir         string
	RandomGenerator *utils.RandomGenerator
	CryptoUtils     *utils.CryptoUtils
}

// NewConfigInitializer 創建新的配置初始化器
func NewConfigInitializer() *ConfigInitializer {
	return &ConfigInitializer{
		RandomGenerator: utils.NewRandomGenerator(),
		CryptoUtils:     utils.NewCryptoUtils(),
	}
}

// SetBaseDir 設置基礎目錄
func (c *ConfigInitializer) SetBaseDir(baseDir string) {
	c.BaseDir = baseDir
}

// Initialize 檢查並初始化配置文件
func (c *ConfigInitializer) Initialize() error {
	// 檢查 BaseDir 是否已設置
	if c.BaseDir == "" {
		// 如果未設置，使用當前工作目錄
		workDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		c.BaseDir = workDir
	}

	// 確保配置目錄存在
	configDirPath := filepath.Join(c.BaseDir, ConfigDir)
	if err := os.MkdirAll(configDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 創建基本目錄結構
	if err := c.initBaseDirs(); err != nil {
		return err
	}

	// 初始化各個配置文件
	if err := c.initCommands(configDirPath); err != nil {
		return err
	}

	if err := c.initConfig(configDirPath); err != nil {
		return err
	}

	if err := c.initPlugins(configDirPath); err != nil {
		return err
	}

	if err := c.initThemes(configDirPath); err != nil {
		return err
	}

	if err := c.initUsers(configDirPath); err != nil {
		return err
	}

	log.Println("Configuration initialization completed")
	return nil
}

// initBaseDirs 創建基本目錄結構
func (c *ConfigInitializer) initBaseDirs() error {
	// 創建 commands 目錄
	commandsDirPath := filepath.Join(c.BaseDir, CommandsDir)
	if err := os.MkdirAll(commandsDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}
	log.Printf("Commands directory created: %s", commandsDirPath)

	// 創建 plugins 目錄
	pluginsDirPath := filepath.Join(c.BaseDir, PluginsDir)
	if err := os.MkdirAll(pluginsDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}
	log.Printf("Plugins directory created: %s", pluginsDirPath)

	// 創建 web 目錄
	webDirPath := filepath.Join(c.BaseDir, WebDir)
	if err := os.MkdirAll(webDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create web directory: %w", err)
	}
	log.Printf("Web directory created: %s", webDirPath)

	return nil
}

// initCommands 初始化命令配置文件
func (c *ConfigInitializer) initCommands(configDir string) error {
	path := filepath.Join(configDir, CommandsFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Command config file %s does not exist, creating empty file", path)

		// 創建空的命令配置
		config := &models.SystemCommandsConfig{
			Enabled: false,
			Routes:  make(map[string]models.CommandConfig),
		}

		// 保存配置
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize command config: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to save command config: %w", err)
		}
	}

	return nil
}

// initConfig 初始化主配置文件
func (c *ConfigInitializer) initConfig(configDir string) error {
	path := filepath.Join(configDir, ConfigYamlFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Main config file %s does not exist, creating default config", path)

		// 尋找可用端口
		port, err := c.RandomGenerator.FindAvailablePort(MinPort, MaxPort, 10)
		if err != nil {
			// 如果找不到可用端口，使用默認端口
			port = 8080
			log.Printf("Could not find an available port, using default port %d", port)
		}

		// 創建默認配置
		config := &models.Config{
			Server: models.ServerConfig{
				Host:    "0.0.0.0",
				Port:    port,
				Timeout: 30,
				Mode:    "release",
			},
			Auth: models.AuthConfig{
				JWTExpiration: 24,
				CookieName:    "panelbase_session",
			},
			Logging: models.LoggingConfig{
				Level: "info",
				File:  "logs/panelbase.log",
			},
			Plugins: models.PluginsConfig{
				Enabled:   true,
				AutoStart: true,
			},
			Routes: models.SystemCommandsConfig{
				Enabled: true,
				Routes:  make(map[string]models.CommandConfig),
			},
		}

		// 保存配置
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to serialize main config: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to save main config: %w", err)
		}
	}

	return nil
}

// initPlugins 初始化插件配置文件
func (c *ConfigInitializer) initPlugins(configDir string) error {
	path := filepath.Join(configDir, PluginsFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Plugin config file %s does not exist, creating empty file", path)

		// 創建空的插件配置
		config := &models.PluginsConfigJSON{
			Plugins: make(map[string]models.PluginInfo),
		}

		// 保存配置
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize plugin config: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to save plugin config: %w", err)
		}
	}

	return nil
}

// initThemes 初始化主題配置文件
func (c *ConfigInitializer) initThemes(configDir string) error {
	path := filepath.Join(configDir, ThemesFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Theme config file %s does not exist, creating default config", path)

		// 從主題模板中讀取
		config := &models.ThemesConfig{
			CurrentTheme: "default_theme",
			Themes: map[string]models.ThemeInfo{
				"default_theme": {
					Name:        "Default Theme",
					Authors:     "PanelBase Team",
					Version:     "1.0.0",
					Description: "Default theme for PanelBase",
					SourceLink:  "https://github.com/OG-Open-Source/PanelBase",
					Directory:   "default",
				},
			},
		}

		// 保存配置
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize theme config: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to save theme config: %w", err)
		}
	}

	return nil
}

// initUsers 初始化用戶配置文件
func (c *ConfigInitializer) initUsers(configDir string) error {
	path := filepath.Join(configDir, UsersFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("User config file %s does not exist, creating default admin user", path)

		// 生成隨機用戶ID
		userID := c.RandomGenerator.GenerateUserID()
		// 生成JWT密鑰
		jwtSecret := c.RandomGenerator.GenerateJWTSecret()
		// 加密默認密碼
		hashedPassword := c.CryptoUtils.HashPassword("admin")

		// 創建默認配置
		config := &models.UsersConfig{
			Users: map[string]*models.User{
				userID: {
					// ID 欄位已被移除
					IsActive:  true,
					Username:  "admin",
					Password:  hashedPassword,
					Role:      "admin",
					Name:      "Administrator",
					Email:     "admin@example.com",
					CreatedAt: models.JsonTime(time.Now()),
					LastLogin: models.JsonTime(time.Time{}), // 使用零值表示從未登入
					API:       make(map[string]models.APIToken),
				},
			},
			DefaultRole: "user",
			JWTSecret:   jwtSecret,
		}

		// 保存配置
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize user config: %w", err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to save user config: %w", err)
		}
	}

	return nil
}
