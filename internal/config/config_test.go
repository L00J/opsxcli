package config

import (
	"os"
	"path/filepath"
	"testing"
)

func resetConfig() {
	globalConfig = nil
	configPath = ""
}

func TestDefaultConfig(t *testing.T) {
	resetConfig()
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if cfg.MySQL.DefaultPort != 3306 {
		t.Errorf("MySQL.DefaultPort = %d, want 3306", cfg.MySQL.DefaultPort)
	}
	if cfg.PostgreSQL.DefaultPort != 5432 {
		t.Errorf("PostgreSQL.DefaultPort = %d, want 5432", cfg.PostgreSQL.DefaultPort)
	}
	if cfg.Redis.DefaultPort != 6379 {
		t.Errorf("Redis.DefaultPort = %d, want 6379", cfg.Redis.DefaultPort)
	}
	if cfg.SSH.Timeout != 30 {
		t.Errorf("SSH.Timeout = %d, want 30", cfg.SSH.Timeout)
	}
	if cfg.Global.LogLevel != "info" {
		t.Errorf("Global.LogLevel = %q, want 'info'", cfg.Global.LogLevel)
	}
}

func TestGet_ReturnsDefaultWhenNil(t *testing.T) {
	resetConfig()
	cfg := Get()
	if cfg == nil {
		t.Fatal("Get returned nil")
	}
	if cfg.MySQL.DefaultPort != 3306 {
		t.Errorf("Get() should return default when globalConfig is nil")
	}
}

func TestSet(t *testing.T) {
	resetConfig()
	newCfg := &Config{
		MySQL: MySQLConfig{DefaultHost: "localhost", DefaultPort: 3307},
	}
	Set(newCfg)
	got := Get()
	if got.MySQL.DefaultPort != 3307 {
		t.Errorf("Get().MySQL.DefaultPort = %d, want 3307", got.MySQL.DefaultPort)
	}
}

func TestLoad_Save_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath = filepath.Join(tmpDir, "config.json")

	cfg := &Config{
		MySQL: MySQLConfig{DefaultHost: "testhost", DefaultPort: 3308},
		Global: GlobalConfig{Debug: true},
	}
	Set(cfg)

	if err := Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// 清空并重新加载
	Set(nil)
	if err := Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	got := Get()
	if got.MySQL.DefaultHost != "testhost" {
		t.Errorf("MySQL.DefaultHost = %q, want 'testhost'", got.MySQL.DefaultHost)
	}
	if got.MySQL.DefaultPort != 3308 {
		t.Errorf("MySQL.DefaultPort = %d, want 3308", got.MySQL.DefaultPort)
	}
	if !got.Global.Debug {
		t.Error("Global.Debug should be true")
	}
}

func TestLoad_CreatesDefaultIfNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	configPath = filepath.Join(tmpDir, "nonexistent.json")

	err := Load()
	if err != nil {
		t.Errorf("Load() should not error when file doesn't exist: %v", err)
	}
	if Get().MySQL.DefaultPort != 3306 {
		t.Error("should load default config when file doesn't exist")
	}

	// 文件应该被创建
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Save should be called to create default config file")
	}
}

func TestSave_InvalidPath(t *testing.T) {
	resetConfig()
	configPath = "/nonexistent/path/config.json"
	cfg := &Config{MySQL: MySQLConfig{DefaultPort: 9999}}
	Set(cfg)

	err := Save()
	if err == nil {
		t.Error("Save should fail for invalid path")
	}
}
