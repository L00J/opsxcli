package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Protection 代码保护模块
type Protection struct {
	checksum string
	enabled  bool
}

// NewProtection 创建保护实例
func NewProtection() *Protection {
	// 计算运行时校验和
	checksum := calculateRuntimeChecksum()

	return &Protection{
		checksum: checksum,
		enabled:  true,
	}
}

// calculateRuntimeChecksum 计算运行时校验和
func calculateRuntimeChecksum() string {
	// 基于运行时环境生成校验和
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	timestamp := time.Now().Unix()

	data := fmt.Sprintf("%s:%d:%d:%s", hostname, pid, timestamp, runtime.Version())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // 取前16字节
}

// Verify 验证运行环境（纯本地验证，支持离线/内网环境）
func (p *Protection) Verify() error {
	if !p.enabled {
		return nil
	}

	// 注意：调试检测在内网环境可能过于严格
	// 如果需要在内网环境调试，可以设置环境变量 OPSXCLI_ALLOW_DEBUG=1 来跳过
	if os.Getenv("OPSXCLI_ALLOW_DEBUG") == "" {
		// 基础验证：检查是否在调试环境中
		if isDebugging() {
			return fmt.Errorf("debugging detected (set OPSXCLI_ALLOW_DEBUG=1 to allow)")
		}
	}

	// 可以添加更多本地验证逻辑（无需网络）
	return nil
}

// isDebugging 检测是否在调试环境中
func isDebugging() bool {
	// 检查常见调试器环境变量
	debugVars := []string{
		"GDB",
		"LLDB",
		"DELVE",
		"DLV",
	}

	for _, env := range debugVars {
		if os.Getenv(env) != "" {
			return true
		}
	}

	return false
}

// GetChecksum 获取校验和
func (p *Protection) GetChecksum() string {
	return p.checksum
}

// AntiTamper 防篡改检查
func (p *Protection) AntiTamper() error {
	// 可以添加二进制完整性检查
	// 例如：检查文件修改时间、大小等

	return nil
}
