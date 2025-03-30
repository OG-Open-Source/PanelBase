package services

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/OG-Open-Source/PanelBase/internal/utils"
)

// ConfigService 提供配置管理功能
type ConfigService struct {
	BaseDir        string // 应用基础目录
	Config         *models.Config
	UsersConfig    *models.UsersConfig
	ThemesConfig   *models.ThemesConfig
	CommandsConfig *models.SystemCommandsConfig
	PluginsConfig  *models.PluginsConfigJSON
}

// NewConfigService 创建新的配置服务
func NewConfigService(configPath string) (*ConfigService, error) {
	// 从配置文件路径获取基础目录
	baseDir, err := utils.GetBaseDir(configPath)
	if err != nil {
		return nil, fmt.Errorf("获取基础目录失败: %w", err)
	}

	service := &ConfigService{
		BaseDir: baseDir,
	}

	// 加载配置文件
	config, err := utils.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %w", err)
	}
	service.Config = config

	// 加载用户配置
	usersPath := filepath.Join(baseDir, utils.UsersFile)
	usersConfig, err := utils.LoadUsersConfig(usersPath)
	if err != nil {
		return nil, fmt.Errorf("加载用户配置失败: %w", err)
	}
	service.UsersConfig = usersConfig

	// 加载主题配置
	themesPath := filepath.Join(baseDir, utils.ThemesFile)
	themesConfig, err := utils.LoadThemesConfig(themesPath)
	if err != nil {
		return nil, fmt.Errorf("加载主题配置失败: %w", err)
	}
	service.ThemesConfig = themesConfig

	// 加载命令配置
	commandsPath := filepath.Join(baseDir, utils.CommandsFile)
	commandsConfig, err := utils.LoadCommandsConfig(commandsPath)
	if err != nil {
		return nil, fmt.Errorf("加载命令配置失败: %w", err)
	}
	service.CommandsConfig = commandsConfig

	// 加载插件配置
	if err := service.LoadPluginsConfig(); err != nil {
		return nil, fmt.Errorf("加载插件配置失败: %w", err)
	}

	return service, nil
}

// GetThemesConfigPath 获取主题配置文件的完整路径
func (s *ConfigService) GetThemesConfigPath() string {
	return filepath.Join(s.BaseDir, utils.ThemesFile)
}

// SaveThemesConfig 保存主题配置到配置文件
func (s *ConfigService) SaveThemesConfig() error {
	return utils.SaveThemesConfig(s.GetThemesConfigPath(), s.ThemesConfig)
}

// GetBaseDir 获取基础目录路径
func GetBaseDir(configPath string) (string, error) {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}

	// 假设配置文件在 configs 目录下，取其父目录
	return filepath.Dir(filepath.Dir(absPath)), nil
}

// GetPluginsConfigPath returns the full path of the plugins configuration file
func (s *ConfigService) GetPluginsConfigPath() string {
	return filepath.Join(s.BaseDir, utils.PluginsFile)
}

// LoadPluginsConfig loads the plugins configuration from JSON file
func (s *ConfigService) LoadPluginsConfig() error {
	pluginsFile := s.GetPluginsConfigPath()
	log.Printf("Loading plugins configuration from ConfigService: %s", pluginsFile)

	config, err := utils.LoadPluginsConfig(pluginsFile)
	if err != nil {
		return fmt.Errorf("failed to load plugins configuration: %w", err)
	}

	s.PluginsConfig = config
	return nil
}

// SavePluginsConfig saves the plugins configuration to JSON file
func (s *ConfigService) SavePluginsConfig() error {
	pluginsFile := s.GetPluginsConfigPath()

	return utils.SavePluginsConfig(pluginsFile, s.PluginsConfig)
}
