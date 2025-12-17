package db

import (
	"opsxcli/internal/config"
)

// ApplyDefaults 应用数据库默认配置
func ApplyDefaults(host string, port int, user string, dbType string) (string, int, string) {
	cfg := config.Get()

	if host == "" {
		switch dbType {
		case "mysql":
			host = cfg.MySQL.DefaultHost
		case "postgres":
			host = cfg.PostgreSQL.DefaultHost
		case "redis":
			host = cfg.Redis.DefaultHost
		}
	}

	if port == 0 {
		switch dbType {
		case "mysql":
			port = cfg.MySQL.DefaultPort
		case "postgres":
			port = cfg.PostgreSQL.DefaultPort
		case "redis":
			port = cfg.Redis.DefaultPort
		}
	}

	if user == "" {
		switch dbType {
		case "mysql":
			user = cfg.MySQL.DefaultUser
		case "postgres":
			user = cfg.PostgreSQL.DefaultUser
		}
	}

	return host, port, user
}
