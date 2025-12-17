/*
 * @Author: Logan.Li
 * @Gitee: https://gitee.com/attacker
 * @email: admin@attacker.club
 * @Date: 2025-12-15 21:21:40
 * @LastEditTime: 2025-12-15 21:45:27
 * @Description:
 */
package core

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Engine 核心引擎，包含关键业务逻辑
type Engine struct {
	version   string
	buildTime string
	gitCommit string
	signature string
}

// NewEngine 创建核心引擎实例
func NewEngine(version, buildTime, gitCommit string) *Engine {
	// 生成唯一签名，用于验证和追踪
	signature := generateSignature(version, buildTime, gitCommit)

	return &Engine{
		version:   version,
		buildTime: buildTime,
		gitCommit: gitCommit,
		signature: signature,
	}
}

// generateSignature 生成唯一签名（用于代码追踪）
func generateSignature(version, buildTime, gitCommit string) string {
	hostname, _ := os.Hostname()
	platform := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	data := fmt.Sprintf("%s:%s:%s:%s:%s", version, buildTime, gitCommit, hostname, platform)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Validate 验证引擎完整性
func (e *Engine) Validate() error {
	// 基础验证逻辑
	if e.version == "" {
		return fmt.Errorf("invalid engine version")
	}

	// 可以添加更多验证逻辑
	// 例如：检查运行环境、验证签名等

	return nil
}

// GetSignature 获取引擎签名（用于追踪）
func (e *Engine) GetSignature() string {
	return e.signature
}

// ProcessData 核心数据处理逻辑（关键业务逻辑放在这里）
func (e *Engine) ProcessData(data interface{}) (interface{}, error) {
	// 关键业务逻辑应该放在 internal/core 中
	// 而不是在 plugins 中，这样可以防止直接查看核心算法

	// 示例：数据加密/处理逻辑
	// 这里可以包含你的核心算法

	return data, nil
}

// CheckLicense 检查许可证（可选）
func (e *Engine) CheckLicense() bool {
	// 可以添加许可证检查逻辑
	// 例如：检查环境变量、配置文件等

	// 暂时返回 true，后续可以扩展
	return true
}

// GetBuildInfo 获取构建信息
func (e *Engine) GetBuildInfo() map[string]string {
	return map[string]string{
		"version":   e.version,
		"buildTime": e.buildTime,
		"gitCommit": e.gitCommit,
		"platform":  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		"goVersion": runtime.Version(),
		"signature": e.signature,
	}
}

// Watermark 添加水印信息（用于追踪代码来源）
func (e *Engine) Watermark() string {
	// 生成水印，可以嵌入到输出中
	timestamp := time.Now().Format("20060102150405")
	return fmt.Sprintf("opsxcli-%s-%s", e.version, timestamp)
}
