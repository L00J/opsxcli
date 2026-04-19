package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 全局配置结构
type Config struct {
	SSH        SSHConfig        `json:"ssh"`
	MySQL      MySQLConfig      `json:"mysql"`
	PostgreSQL PostgreSQLConfig `json:"postgresql"`
	Redis      RedisConfig      `json:"redis"`
	Global     GlobalConfig     `json:"global"`
}

// SSHConfig SSH配置
type SSHConfig struct {
	DefaultKeyPath string `json:"default_key_path"`
	Timeout        int    `json:"timeout"` // 秒
}

// MySQLConfig MySQL配置
type MySQLConfig struct {
	DefaultHost string `json:"default_host"`
	DefaultPort int    `json:"default_port"`
	DefaultUser string `json:"default_user"`
}

// PostgreSQLConfig PostgreSQL配置
type PostgreSQLConfig struct {
	DefaultHost string `json:"default_host"`
	DefaultPort int    `json:"default_port"`
	DefaultUser string `json:"default_user"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	DefaultHost string `json:"default_host"`
	DefaultPort int    `json:"default_port"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	Debug    bool   `json:"debug"`
	LogLevel string `json:"log_level"`
	AutoSave bool   `json:"auto_save"`
}

// ═══════════════════════════════════════════════════════════════
// Manager — 配置管理器（支持依赖注入，替代全局变量）
// ═══════════════════════════════════════════════════════════════

// Manager 配置管理器，封装配置状态，支持独立实例
type Manager struct {
	config *Config
	path   string
}

// NewManager 创建独立的配置管理器实例
// configDir: 配置文件目录，空字符串使用默认值 ~/.opsxcli
func NewManager(configDir string) (*Manager, error) {
	if configDir == "" {
		homeDir := os.Getenv("HOME")
		configDir = filepath.Join(homeDir, ".opsxcli")
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	m := &Manager{
		path: filepath.Join(configDir, "config.json"),
	}

	// 尝试加载配置，不存在则使用默认
	if err := m.Load(); err != nil {
		m.config = DefaultConfig()
	}

	return m, nil
}

// Load 加载配置文件
func (m *Manager) Load() error {
	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		m.config = DefaultConfig()
		return m.Save()
	}

	data, err := os.ReadFile(m.path)
	if err != nil {
		m.config = DefaultConfig()
		return err
	}

	if err := json.Unmarshal(data, &m.config); err != nil {
		m.config = DefaultConfig()
		return err
	}

	return nil
}

// Save 保存配置文件
func (m *Manager) Save() error {
	if m.config == nil {
		m.config = DefaultConfig()
	}
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}

// Get 获取配置
func (m *Manager) Get() *Config {
	if m.config == nil {
		m.config = DefaultConfig()
	}
	return m.config
}

// Set 设置配置
func (m *Manager) Set(config *Config) {
	m.config = config
}

// Path 返回配置文件路径
func (m *Manager) Path() string {
	return m.path
}

// ═══════════════════════════════════════════════════════════════
// 包级兼容层 — 保留原有 API，委托给默认 Manager 实例
// ═══════════════════════════════════════════════════════════════

var (
	globalConfig *Config
	configPath   string
	defaultMgr   *Manager // 内部使用的默认管理器实例
)

// Init 初始化配置系统（兼容层）
func Init() error {
	homeDir := os.Getenv("HOME")
	configDir := filepath.Join(homeDir, ".opsxcli")
	configPath = filepath.Join(configDir, "config.json")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 同步初始化默认 Manager
	var err error
	defaultMgr, err = NewManager(configDir)
	if err != nil {
		return err
	}
	globalConfig = defaultMgr.Get()

	return nil
}

// Load 加载配置文件（兼容层）
func Load() error {
	if defaultMgr != nil {
		err := defaultMgr.Load()
		globalConfig = defaultMgr.Get()
		configPath = defaultMgr.Path()
		return err
	}

	// 回退到传统逻辑（避免 nil pointer）
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		globalConfig = DefaultConfig()
		return Save()
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		globalConfig = DefaultConfig()
		return err
	}

	if err := json.Unmarshal(data, &globalConfig); err != nil {
		globalConfig = DefaultConfig()
		return err
	}

	return nil
}

// Save 保存配置文件（兼容层）
func Save() error {
	if defaultMgr != nil {
		return defaultMgr.Save()
	}
	data, err := json.MarshalIndent(globalConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		SSH: SSHConfig{
			DefaultKeyPath: filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
			Timeout:        30,
		},
		MySQL: MySQLConfig{
			DefaultHost: "127.0.0.1",
			DefaultPort: 3306,
			DefaultUser: "root",
		},
		PostgreSQL: PostgreSQLConfig{
			DefaultHost: "127.0.0.1",
			DefaultPort: 5432,
			DefaultUser: "postgres",
		},
		Redis: RedisConfig{
			DefaultHost: "127.0.0.1",
			DefaultPort: 6379,
		},
		Global: GlobalConfig{
			Debug:    false,
			LogLevel: "info",
			AutoSave: true,
		},
	}
}

// Get 获取全局配置（兼容层）
func Get() *Config {
	if defaultMgr != nil {
		return defaultMgr.Get()
	}
	if globalConfig == nil {
		globalConfig = DefaultConfig()
	}
	return globalConfig
}

// Set 设置全局配置（兼容层）
func Set(config *Config) {
	if defaultMgr != nil {
		defaultMgr.Set(config)
	}
	globalConfig = config
}
