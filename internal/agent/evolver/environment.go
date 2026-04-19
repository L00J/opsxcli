// environment.go - 环境记忆系统
// Layer 3: 环境记忆 - 记录服务器特征、用户偏好、常用配置
// 存储: ~/.opsxcli/agent/environment.json
package evolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// KnownServer 已知服务器记录
type KnownServer struct {
	Host         string            `json:"host"`           // 主机地址 user@host:port
	OS           string            `json:"os"`             // 操作系统（自动检测）
	DefaultUser  string            `json:"default_user"`   // 默认用户名
	CommonPaths  []string          `json:"common_paths"`   // 常用路径
	LastUsed     time.Time         `json:"last_used"`      // 最后使用时间
	UseCount     int               `json:"use_count"`      // 使用次数
	Tags         []string          `json:"tags"`           // 标签（如 "生产环境"、"测试环境"）
	CustomInfo   map[string]string `json:"custom_info"`    // 自定义信息
}

// UserPreference 用户偏好
type UserPreference struct {
	SafetyMode       string   `json:"safety_mode"`        // 默认安全模式
	AutoApproveLow   bool     `json:"auto_approve_low"`   // 自动批准低风险操作
	PreferSudo       bool     `json:"prefer_sudo"`        // 默认使用 sudo
	TimeoutSeconds   int      `json:"timeout_seconds"`    // 默认超时
	Editor           string   `json:"editor"`             // 首选编辑器
	Shell            string   `json:"shell"`              // 首选 shell
}

// EnvironmentMemory 环境记忆
type EnvironmentMemory struct {
	mu           sync.RWMutex
	KnownServers []*KnownServer    `json:"known_servers"`   // 已知服务器列表
	UserPrefs    UserPreference    `json:"user_preferences"` // 用户偏好
	LastQueries  []string          `json:"last_queries"`     // 最近查询（去重，最多 50 条）
	CustomHints  map[string]string `json:"custom_hints"`     // 用户自定义提示
	baseDir      string            // 存储目录
}

// NewEnvironmentMemory 创建环境记忆
func NewEnvironmentMemory(baseDir string) *EnvironmentMemory {
	return &EnvironmentMemory{
		KnownServers: make([]*KnownServer, 0),
		UserPrefs: UserPreference{
			SafetyMode:     "balanced",
			AutoApproveLow: false,
			PreferSudo:     false,
			TimeoutSeconds: 60,
			Editor:         "vim",
			Shell:          "bash",
		},
		LastQueries: make([]string, 0),
		CustomHints: make(map[string]string),
		baseDir:     baseDir,
	}
}

// LoadEnvironmentMemory 从磁盘加载环境记忆
func LoadEnvironmentMemory(baseDir string) (*EnvironmentMemory, error) {
	filePath := filepath.Join(baseDir, "environment.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewEnvironmentMemory(baseDir), nil
		}
		return nil, fmt.Errorf("读取环境文件失败: %w", err)
	}

	var em EnvironmentMemory
	if err := json.Unmarshal(data, &em); err != nil {
		return nil, fmt.Errorf("解析环境文件失败: %w", err)
	}

	em.baseDir = baseDir
	if em.LastQueries == nil {
		em.LastQueries = make([]string, 0)
	}
	if em.CustomHints == nil {
		em.CustomHints = make(map[string]string)
	}
	if em.KnownServers == nil {
		em.KnownServers = make([]*KnownServer, 0)
	}

	return &em, nil
}

// Save 保存到磁盘
func (em *EnvironmentMemory) Save() error {
	em.mu.Lock()
	defer em.mu.Unlock()

	filePath := filepath.Join(em.baseDir, "environment.json")
	data, err := json.MarshalIndent(em, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化环境记忆失败: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("写入环境文件失败: %w", err)
	}

	return nil
}

// RecordServer 记录/更新服务器信息
func (em *EnvironmentMemory) RecordServer(host string, info map[string]string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 解析 host
	user, hostname, port := em.parseHost(host)

	// 查找是否已存在
	for _, s := range em.KnownServers {
		if s.Host == hostname && s.DefaultUser == user {
			s.LastUsed = time.Now()
			s.UseCount++
			// 更新自定义信息
			if s.CustomInfo == nil {
				s.CustomInfo = make(map[string]string)
			}
			for k, v := range info {
				s.CustomInfo[k] = v
			}
			// 更新 OS（如果检测到了）
			if osName, ok := info["os"]; ok && osName != "" {
				s.OS = osName
			}
			// 追加常用路径
			if path, ok := info["path"]; ok && path != "" {
				s.CommonPaths = em.appendUnique(s.CommonPaths, path)
			}
			if port != "" && port != "22" {
				if s.CustomInfo == nil {
					s.CustomInfo = make(map[string]string)
				}
				s.CustomInfo["port"] = port
			}
			return
		}
	}

	// 新增服务器
	newServer := &KnownServer{
		Host:        hostname,
		DefaultUser: user,
		OS:          info["os"],
		LastUsed:    time.Now(),
		UseCount:    1,
		Tags:        make([]string, 0),
		CustomInfo:  make(map[string]string),
	}
	for k, v := range info {
		newServer.CustomInfo[k] = v
	}
	if port != "" && port != "22" {
		newServer.CustomInfo["port"] = port
	}

	em.KnownServers = append(em.KnownServers, newServer)
}

// GetServer 获取服务器信息
func (em *EnvironmentMemory) GetServer(host string) *KnownServer {
	em.mu.RLock()
	defer em.mu.RUnlock()

	_, hostname, _ := em.parseHost(host)
	for _, s := range em.KnownServers {
		if s.Host == hostname {
			return s
		}
	}
	return nil
}

// GetRecentServers 获取最近使用的服务器（按时间倒序）
func (em *EnvironmentMemory) GetRecentServers(limit int) []*KnownServer {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if limit <= 0 || limit > len(em.KnownServers) {
		limit = len(em.KnownServers)
	}

	// 复制并排序
	servers := make([]*KnownServer, len(em.KnownServers))
	copy(servers, em.KnownServers)

	// 按 LastUsed 倒序
	for i := 0; i < len(servers)-1; i++ {
		for j := i + 1; j < len(servers); j++ {
			if servers[j].LastUsed.After(servers[i].LastUsed) {
				servers[i], servers[j] = servers[j], servers[i]
			}
		}
	}

	return servers[:limit]
}

// GetServerSuggestions 获取服务器建议（自动补全用）
func (em *EnvironmentMemory) GetServerSuggestions(prefix string) []string {
	em.mu.RLock()
	defer em.mu.RUnlock()

	suggestions := make([]string, 0)
	for _, s := range em.KnownServers {
		candidate := s.DefaultUser + "@" + s.Host
		if strings.HasPrefix(candidate, prefix) || strings.HasPrefix(s.Host, prefix) {
			suggestions = append(suggestions, candidate)
		}
	}
	return suggestions
}

// UpdateLastQueries 更新最近查询
func (em *EnvironmentMemory) UpdateLastQueries(query string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	// 去重检查
	for i, q := range em.LastQueries {
		if q == query {
			// 移到末尾（最新）
			em.LastQueries = append(em.LastQueries[:i], em.LastQueries[i+1:]...)
			em.LastQueries = append(em.LastQueries, query)
			return
		}
	}

	em.LastQueries = append(em.LastQueries, query)

	// 限制最多 50 条
	if len(em.LastQueries) > 50 {
		em.LastQueries = em.LastQueries[len(em.LastQueries)-50:]
	}
}

// GetSimilarQueries 获取相似查询（用于提示）
func (em *EnvironmentMemory) GetSimilarQueries(query string, limit int) []string {
	em.mu.RLock()
	defer em.mu.RUnlock()

	type scored struct {
		query string
		score float64
	}

	scoredList := make([]scored, 0)
	for _, q := range em.LastQueries {
		if q == query {
			continue
		}
		s := em.querySimilarity(query, q)
		if s > 0.3 { // 相似度阈值
			scoredList = append(scoredList, scored{query: q, score: s})
		}
	}

	// 按相似度排序
	for i := 0; i < len(scoredList)-1; i++ {
		for j := i + 1; j < len(scoredList); j++ {
			if scoredList[j].score > scoredList[i].score {
				scoredList[i], scoredList[j] = scoredList[j], scoredList[i]
			}
		}
	}

	// 取前 limit 个
	result := make([]string, 0)
	count := limit
	if count > len(scoredList) {
		count = len(scoredList)
	}
	for i := 0; i < count; i++ {
		result = append(result, scoredList[i].query)
	}
	return result
}

// SetUserPreference 设置用户偏好
func (em *EnvironmentMemory) SetUserPreference(key, value string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	switch key {
	case "safety_mode":
		em.UserPrefs.SafetyMode = value
	case "auto_approve_low":
		em.UserPrefs.AutoApproveLow = value == "true"
	case "prefer_sudo":
		em.UserPrefs.PreferSudo = value == "true"
	case "timeout":
		// 尝试解析为整数
		var sec int
		fmt.Sscanf(value, "%d", &sec)
		if sec > 0 {
			em.UserPrefs.TimeoutSeconds = sec
		}
	case "editor":
		em.UserPrefs.Editor = value
	case "shell":
		em.UserPrefs.Shell = value
	}
}

// GetUserPreference 获取用户偏好
func (em *EnvironmentMemory) GetUserPreference(key string) string {
	em.mu.RLock()
	defer em.mu.RUnlock()

	switch key {
	case "safety_mode":
		return em.UserPrefs.SafetyMode
	case "auto_approve_low":
		if em.UserPrefs.AutoApproveLow {
			return "true"
		}
		return "false"
	case "prefer_sudo":
		if em.UserPrefs.PreferSudo {
			return "true"
		}
		return "false"
	case "timeout":
		return fmt.Sprintf("%d", em.UserPrefs.TimeoutSeconds)
	case "editor":
		return em.UserPrefs.Editor
	case "shell":
		return em.UserPrefs.Shell
	default:
		return ""
	}
}

// AddCustomHint 添加用户自定义提示
func (em *EnvironmentMemory) AddCustomHint(taskType, hint string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.CustomHints == nil {
		em.CustomHints = make(map[string]string)
	}
	em.CustomHints[taskType] = hint
}

// GetCustomHint 获取用户自定义提示
func (em *EnvironmentMemory) GetCustomHint(taskType string) string {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if em.CustomHints == nil {
		return ""
	}
	return em.CustomHints[taskType]
}

// RemoveServer 删除服务器记录
func (em *EnvironmentMemory) RemoveServer(host string) bool {
	em.mu.Lock()
	defer em.mu.Unlock()

	_, hostname, _ := em.parseHost(host)
	for i, s := range em.KnownServers {
		if s.Host == hostname {
			em.KnownServers = append(em.KnownServers[:i], em.KnownServers[i+1:]...)
			return true
		}
	}
	return false
}

// ServerCount 获取服务器数量
func (em *EnvironmentMemory) ServerCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return len(em.KnownServers)
}

// parseHost 解析 host 字符串为 user, hostname, port
func (em *EnvironmentMemory) parseHost(host string) (user, hostname, port string) {
	user = "root"
	port = "22"

	// 格式: user@host:port 或 user@host 或 host
	atIdx := strings.LastIndex(host, "@")
	if atIdx >= 0 {
		user = host[:atIdx]
		host = host[atIdx+1:]
	}

	colonIdx := strings.LastIndex(host, ":")
	if colonIdx >= 0 {
		port = host[colonIdx+1:]
		hostname = host[:colonIdx]
	} else {
		hostname = host
	}

	return
}

// appendUnique 追加唯一元素到切片
func (em *EnvironmentMemory) appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// querySimilarity 计算两个查询的相似度
func (em *EnvironmentMemory) querySimilarity(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// 简单实现：共享词比例
	wordsA := strings.Fields(a)
	wordsB := strings.Fields(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	// 构建词频表
	freqA := make(map[string]bool)
	for _, w := range wordsA {
		freqA[w] = true
	}

	shared := 0
	for _, w := range wordsB {
		if freqA[w] {
			shared++
		}
	}

	// Jaccard 相似度
	union := len(wordsA) + len(wordsB) - shared
	if union == 0 {
		return 0
	}
	return float64(shared) / float64(union)
}
