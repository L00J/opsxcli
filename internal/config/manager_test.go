package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === Manager 测试 ===

func TestNewManager_CustomDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "myconfig")

	mgr, err := NewManager(configDir)
	require.NoError(t, err)
	require.NotNil(t, mgr)

	// 应创建目录
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// 应加载默认配置
	cfg := mgr.Get()
	assert.Equal(t, 3306, cfg.MySQL.DefaultPort)
	assert.Equal(t, 5432, cfg.PostgreSQL.DefaultPort)
	assert.Equal(t, 6379, cfg.Redis.DefaultPort)
	assert.Equal(t, 30, cfg.SSH.Timeout)

	// 路径应正确
	assert.Equal(t, filepath.Join(configDir, "config.json"), mgr.Path())
}

func TestNewManager_EmptyDir(t *testing.T) {
	// 空字符串应使用默认 ~/.opsxcli
	mgr, err := NewManager("")
	require.NoError(t, err)
	require.NotNil(t, mgr)

	expectedPath := filepath.Join(os.Getenv("HOME"), ".opsxcli", "config.json")
	assert.Equal(t, expectedPath, mgr.Path())
}

func TestManager_Load_Save_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 修改配置
	cfg := mgr.Get()
	cfg.MySQL.DefaultHost = "custom-host"
	cfg.MySQL.DefaultPort = 3307
	cfg.Global.Debug = true
	cfg.Global.LogLevel = "debug"

	// 保存
	err = mgr.Save()
	require.NoError(t, err)

	// 创建新 manager 加载同一目录
	mgr2, err := NewManager(tmpDir)
	require.NoError(t, err)

	cfg2 := mgr2.Get()
	assert.Equal(t, "custom-host", cfg2.MySQL.DefaultHost)
	assert.Equal(t, 3307, cfg2.MySQL.DefaultPort)
	assert.True(t, cfg2.Global.Debug)
	assert.Equal(t, "debug", cfg2.Global.LogLevel)
}

func TestManager_Load_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// 写入无效 JSON
	err := os.WriteFile(configFile, []byte("{invalid json}"), 0644)
	require.NoError(t, err)

	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 应回退到默认配置
	cfg := mgr.Get()
	assert.Equal(t, 3306, cfg.MySQL.DefaultPort)
}

func TestManager_Load_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	// 写入有效 JSON 但设为不可读
	err := os.WriteFile(configFile, []byte(`{"mysql":{"default_port":3306}}`), 0000)
	require.NoError(t, err)
	defer os.Chmod(configFile, 0644)

	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 加载应回退到默认配置
	cfg := mgr.Get()
	assert.NotNil(t, cfg)
}

func TestManager_Get_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 手动设置 nil
	mgr.config = nil

	cfg := mgr.Get()
	assert.NotNil(t, cfg)
	assert.Equal(t, 3306, cfg.MySQL.DefaultPort)
}

func TestManager_Set(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	newCfg := &Config{
		MySQL: MySQLConfig{DefaultHost: "newhost", DefaultPort: 9999},
	}
	mgr.Set(newCfg)

	got := mgr.Get()
	assert.Equal(t, "newhost", got.MySQL.DefaultHost)
	assert.Equal(t, 9999, got.MySQL.DefaultPort)
}

func TestManager_Path(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	expected := filepath.Join(tmpDir, "config.json")
	assert.Equal(t, expected, mgr.Path())
}

func TestManager_Save_NilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(tmpDir)
	require.NoError(t, err)

	// 手动设置 nil
	mgr.config = nil

	// Save 应自动使用默认配置
	err = mgr.Save()
	require.NoError(t, err)

	// 验证文件内容是默认配置
	data, err := os.ReadFile(mgr.Path())
	require.NoError(t, err)

	var saved Config
	err = json.Unmarshal(data, &saved)
	require.NoError(t, err)
	assert.Equal(t, 3306, saved.MySQL.DefaultPort)
}

// === Config 结构体序列化测试 ===

func TestConfig_JSONRoundTrip(t *testing.T) {
	cfg := DefaultConfig()

	data, err := json.MarshalIndent(cfg, "", "  ")
	require.NoError(t, err)

	var loaded Config
	err = json.Unmarshal(data, &loaded)
	require.NoError(t, err)

	assert.Equal(t, cfg.SSH.DefaultKeyPath, loaded.SSH.DefaultKeyPath)
	assert.Equal(t, cfg.SSH.Timeout, loaded.SSH.Timeout)
	assert.Equal(t, cfg.MySQL.DefaultHost, loaded.MySQL.DefaultHost)
	assert.Equal(t, cfg.MySQL.DefaultPort, loaded.MySQL.DefaultPort)
	assert.Equal(t, cfg.MySQL.DefaultUser, loaded.MySQL.DefaultUser)
	assert.Equal(t, cfg.PostgreSQL.DefaultHost, loaded.PostgreSQL.DefaultHost)
	assert.Equal(t, cfg.PostgreSQL.DefaultPort, loaded.PostgreSQL.DefaultPort)
	assert.Equal(t, cfg.PostgreSQL.DefaultUser, loaded.PostgreSQL.DefaultUser)
	assert.Equal(t, cfg.Redis.DefaultHost, loaded.Redis.DefaultHost)
	assert.Equal(t, cfg.Redis.DefaultPort, loaded.Redis.DefaultPort)
	assert.Equal(t, cfg.Global.Debug, loaded.Global.Debug)
	assert.Equal(t, cfg.Global.LogLevel, loaded.Global.LogLevel)
	assert.Equal(t, cfg.Global.AutoSave, loaded.Global.AutoSave)
}

// === 包级函数 补充测试 ===

func TestInit(t *testing.T) {
	resetConfig()
	// Init 应能正常执行
	err := Init()
	require.NoError(t, err)
	assert.NotNil(t, defaultMgr)
	assert.NotNil(t, globalConfig)
	// 加载后的配置应有合理的端口值（可能是默认值或已有配置）
	assert.NotZero(t, globalConfig.MySQL.DefaultPort)

	// 再次调用也应成功（幂等）
	err = Init()
	require.NoError(t, err)
}

func TestLoad_Save_WithDefaultMgr(t *testing.T) {
	resetConfig()

	err := Init()
	require.NoError(t, err)

	// 通过包级 Set 修改配置
	newCfg := &Config{
		MySQL:      MySQLConfig{DefaultHost: "loadtest", DefaultPort: 3406},
		PostgreSQL: PostgreSQLConfig{DefaultHost: "pgtest", DefaultPort: 5500},
		Redis:      RedisConfig{DefaultHost: "redistest", DefaultPort: 6380},
		Global:     GlobalConfig{Debug: true, LogLevel: "warn"},
	}
	Set(newCfg)

	err = Save()
	require.NoError(t, err)

	// 清空并重新加载
	globalConfig = nil
	err = Load()
	require.NoError(t, err)

	got := Get()
	assert.Equal(t, "loadtest", got.MySQL.DefaultHost)
	assert.Equal(t, 3406, got.MySQL.DefaultPort)
	assert.Equal(t, "pgtest", got.PostgreSQL.DefaultHost)
	assert.Equal(t, 5500, got.PostgreSQL.DefaultPort)
	assert.Equal(t, "redistest", got.Redis.DefaultHost)
	assert.Equal(t, 6380, got.Redis.DefaultPort)
	assert.True(t, got.Global.Debug)
	assert.Equal(t, "warn", got.Global.LogLevel)
}

// === DefaultConfig 字段完整性测试 ===

func TestDefaultConfig_AllFields(t *testing.T) {
	cfg := DefaultConfig()
	require.NotNil(t, cfg)

	// SSH
	assert.Contains(t, cfg.SSH.DefaultKeyPath, ".ssh")
	assert.Equal(t, 30, cfg.SSH.Timeout)

	// MySQL
	assert.Equal(t, "127.0.0.1", cfg.MySQL.DefaultHost)
	assert.Equal(t, 3306, cfg.MySQL.DefaultPort)
	assert.Equal(t, "root", cfg.MySQL.DefaultUser)

	// PostgreSQL
	assert.Equal(t, "127.0.0.1", cfg.PostgreSQL.DefaultHost)
	assert.Equal(t, 5432, cfg.PostgreSQL.DefaultPort)
	assert.Equal(t, "postgres", cfg.PostgreSQL.DefaultUser)

	// Redis
	assert.Equal(t, "127.0.0.1", cfg.Redis.DefaultHost)
	assert.Equal(t, 6379, cfg.Redis.DefaultPort)

	// Global
	assert.False(t, cfg.Global.Debug)
	assert.Equal(t, "info", cfg.Global.LogLevel)
	assert.True(t, cfg.Global.AutoSave)
}
