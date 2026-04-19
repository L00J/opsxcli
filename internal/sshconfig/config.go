package sshconfig

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Host SSH 主机配置
type Host struct {
	Alias        string // Host 别名（如 "prod", "web1"）
	HostName     string // 实际主机地址
	User         string // 用户名
	Port         int    // 端口（默认 22）
	IdentityFile string // 私钥路径
}

// Config SSH 配置
type Config struct {
	Hosts []Host
}

// DefaultPath 返回默认的 SSH 配置文件路径
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

// Parse 解析指定路径的 SSH 配置文件
func Parse(path string) (*Config, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Hosts: []Host{}}, nil
		}
		return nil, err
	}
	if info.IsDir() {
		return &Config{Hosts: []Host{}}, nil
	}
	return parseFile(path)
}

func parseFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := &Config{Hosts: []Host{}}
	scanner := bufio.NewScanner(file)

	var currentHosts []*Host

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}

		directive := strings.ToLower(fields[0])
		rest := strings.Join(fields[1:], " ")

		switch directive {
		case "host":
			// 保存当前解析中的主机
			for _, h := range currentHosts {
				cfg.Hosts = append(cfg.Hosts, *h)
			}
			currentHosts = nil

			aliases := fields[1:]
			for _, alias := range aliases {
				if isWildcard(alias) {
					continue
				}
				currentHosts = append(currentHosts, &Host{
					Alias: alias,
					Port:  22,
				})
			}
		case "hostname":
			for _, h := range currentHosts {
				h.HostName = rest
			}
		case "user":
			for _, h := range currentHosts {
				h.User = rest
			}
		case "port":
			if p, err := strconv.Atoi(rest); err == nil {
				for _, h := range currentHosts {
					h.Port = p
				}
			}
		case "identityfile":
			filePath := expandPath(unquote(rest))
			for _, h := range currentHosts {
				h.IdentityFile = filePath
			}
		case "include":
			includePattern := expandPath(unquote(rest))
			matches, err := filepath.Glob(includePattern)
			if err == nil {
				for _, m := range matches {
					included, err := parseFile(m)
					if err == nil {
						cfg.Hosts = append(cfg.Hosts, included.Hosts...)
					}
				}
			}
		}
	}

	for _, h := range currentHosts {
		cfg.Hosts = append(cfg.Hosts, *h)
	}

	return cfg, scanner.Err()
}

// GetHost 根据别名获取主机配置
func (c *Config) GetHost(alias string) *Host {
	for i := range c.Hosts {
		if c.Hosts[i].Alias == alias {
			h := c.Hosts[i]
			if h.HostName == "" {
				h.HostName = h.Alias
			}
			return &h
		}
	}
	return nil
}

// ListAliases 返回所有主机别名列表
func (c *Config) ListAliases() []string {
	aliases := make([]string, 0, len(c.Hosts))
	for _, h := range c.Hosts {
		aliases = append(aliases, h.Alias)
	}
	return aliases
}

func isWildcard(alias string) bool {
	return strings.Contains(alias, "*") || strings.Contains(alias, "?")
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	// ssh config 也支持 %d 表示 home dir，简单处理
	if strings.HasPrefix(p, "%d/") || strings.HasPrefix(p, "%d\\") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[3:])
		}
	}
	return p
}
