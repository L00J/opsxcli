package config

import (
	"encoding/json"
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

var (
	globalConfig *Config
	configPath   string
)

// Init 初始化配置系统
func Init() error {
	// 确定配置文件路径
	homeDir := os.Getenv("HOME")
	configDir := filepath.Join(homeDir, ".opsxcli")
	configPath = filepath.Join(configDir, "config.json")

	// 创建配置目录
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 加载配置
	return Load()
}

// Load 加载配置文件
func Load() error {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		globalConfig = DefaultConfig()
		return Save()
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		globalConfig = DefaultConfig()
		return err
	}

	// 解析JSON
	if err := json.Unmarshal(data, &globalConfig); err != nil {
		globalConfig = DefaultConfig()
		return err
	}

	return nil
}

// Save 保存配置文件
func Save() error {
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

// Get 获取全局配置
func Get() *Config {
	if globalConfig == nil {
		globalConfig = DefaultConfig()
	}
	return globalConfig
}

// Set 设置全局配置
func Set(config *Config) {
	globalConfig = config
}
