package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// DB 数据库连接
type DB struct {
	*sql.DB
}

// NewDB 创建数据库连接
func NewDB(dataDir string) (*DB, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	dbPath := filepath.Join(dataDir, "opsxcli.db")

	// 连接数据库
	sqlDB, err := sql.Open("sqlite", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	db := &DB{DB: sqlDB}

	// 初始化 schema
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	return db, nil
}

// initSchema 初始化数据库 schema
func (db *DB) initSchema() error {
	_, err := db.Exec(schemaSQL)
	return err
}

// GetDefaultDataDir 获取默认数据目录
func GetDefaultDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".opsxcli", "data"), nil
}
