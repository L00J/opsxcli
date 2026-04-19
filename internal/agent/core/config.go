// config.go - Agent 配置定义
package core

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// SafetyMode 安全模式类型
type SafetyMode string

const (
	SafetyModeStrict     SafetyMode = "strict"     // 严格模式：medium+ 需要确认
	SafetyModeBalanced   SafetyMode = "balanced"   // 平衡模式：high+ 需要确认（默认）
	SafetyModePermissive SafetyMode = "permissive" // 宽松模式：只有 critical 需要确认
)

// String 返回安全模式的字符串表示
func (s SafetyMode) String() string {
	return string(s)
}

// Config Agent 配置结构体
type Config struct {
	MaxIterations      int           // 最大迭代次数，默认 10
	Temperature        float64       // LLM 温度，默认 0.3
	ToolTimeout        time.Duration // 工具执行超时，默认 60s
	MaxTokens          int           // 最大 token 数，默认 4096
	SafetyMode         SafetyMode    // 安全模式，默认 balanced
	SessionDir         string        // 会话存储目录，默认 ~/.opsxcli/agent/sessions
	AutoApprove        bool          // 自动批准（危险，仅测试），默认 false
	SSHConnectTimeout  time.Duration // SSH 连接超时，默认 10s
	OutputMaxLength    int           // 输出最大长度（超过则截断），默认 10000
	MaxContextTokens   int           // 最大上下文 token 数，默认 6000（为 8k 模型留余量）
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	return &Config{
		MaxIterations:      10,
		Temperature:        0.3,
		ToolTimeout:        60 * time.Second,
		MaxTokens:          4096,
		SafetyMode:         SafetyModeBalanced,
		SessionDir:         filepath.Join(homeDir, ".opsxcli", "agent", "sessions"),
		AutoApprove:        false,
		SSHConnectTimeout:  10 * time.Second,
		OutputMaxLength:    10000,
		MaxContextTokens:   6000,
	}
}

// LoadConfig 从环境变量加载配置，覆盖默认值
func LoadConfig() *Config {
	config := NewDefaultConfig()

	// OPSXCLI_AGENT_MAX_ITERATIONS - 最大迭代次数
	if v := os.Getenv("OPSXCLI_AGENT_MAX_ITERATIONS"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			config.MaxIterations = iv
		}
	}

	// OPSXCLI_AGENT_TEMPERATURE - LLM 温度
	if v := os.Getenv("OPSXCLI_AGENT_TEMPERATURE"); v != "" {
		if fv, err := strconv.ParseFloat(v, 64); err == nil && fv >= 0 && fv <= 2 {
			config.Temperature = fv
		}
	}

	// OPSXCLI_AGENT_TOOL_TIMEOUT - 工具超时（秒）
	if v := os.Getenv("OPSXCLI_AGENT_TOOL_TIMEOUT"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			config.ToolTimeout = time.Duration(iv) * time.Second
		}
	}

	// OPSXCLI_AGENT_MAX_TOKENS - 最大 token 数
	if v := os.Getenv("OPSXCLI_AGENT_MAX_TOKENS"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			config.MaxTokens = iv
		}
	}

	// OPSXCLI_AGENT_SAFETY_MODE - 安全模式
	if v := os.Getenv("OPSXCLI_AGENT_SAFETY_MODE"); v != "" {
		switch v {
		case "strict":
			config.SafetyMode = SafetyModeStrict
		case "balanced":
			config.SafetyMode = SafetyModeBalanced
		case "permissive":
			config.SafetyMode = SafetyModePermissive
		}
	}

	// OPSXCLI_AGENT_SESSION_DIR - 会话目录
	if v := os.Getenv("OPSXCLI_AGENT_SESSION_DIR"); v != "" {
		config.SessionDir = v
	}

	// OPSXCLI_AGENT_AUTO_APPROVE - 自动批准
	if v := os.Getenv("OPSXCLI_AGENT_AUTO_APPROVE"); v != "" {
		if bv, err := strconv.ParseBool(v); err == nil {
			config.AutoApprove = bv
		}
	}

	// OPSXCLI_AGENT_SSH_TIMEOUT - SSH 连接超时（秒）
	if v := os.Getenv("OPSXCLI_AGENT_SSH_TIMEOUT"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			config.SSHConnectTimeout = time.Duration(iv) * time.Second
		}
	}

	// OPSXCLI_AGENT_OUTPUT_MAX_LENGTH - 输出最大长度
	if v := os.Getenv("OPSXCLI_AGENT_OUTPUT_MAX_LENGTH"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			config.OutputMaxLength = iv
		}
	}

	// OPSXCLI_AGENT_MAX_CONTEXT_TOKENS - 最大上下文 token 数
	if v := os.Getenv("OPSXCLI_AGENT_MAX_CONTEXT_TOKENS"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			config.MaxContextTokens = iv
		}
	}

	return config
}
