// execute.go - 统一执行工具（v0.5.0）
// 合并 local_bash + ssh_execute 为统一接口，本地/远程自动路由
// LLM 只需调用一个 "execute" 工具，无需区分本地或远程
package tools

import (
	"context"
	"fmt"
)

// UnifiedExecuteTool 统一执行工具
// 无 host 参数 → 本地执行，有 host 参数 → SSH 远程执行
type UnifiedExecuteTool struct {
	localBash *LocalBashTool
	sshExec   *SSHExecuteTool
}

// NewUnifiedExecuteTool 创建统一执行工具
func NewUnifiedExecuteTool() *UnifiedExecuteTool {
	return &UnifiedExecuteTool{
		localBash: NewLocalBashTool(),
		sshExec:   NewSSHExecuteTool(),
	}
}

// Close 释放资源（实现 Closer 接口）
func (t *UnifiedExecuteTool) Close() error {
	if t.sshExec != nil {
		return t.sshExec.Close()
	}
	return nil
}

// Name 返回工具名称
func (t *UnifiedExecuteTool) Name() string {
	return "execute"
}

// Description 返回工具描述（给 LLM 看的）
func (t *UnifiedExecuteTool) Description() string {
	return `统一命令执行工具。自动路由到本地或远程执行：
- 不指定 host → 本地执行（相当于 local_bash）
- 指定 host → SSH 远程执行（相当于 ssh_execute）
支持所有 bash 命令，包括 grep/awk/sed/cat/systemctl 等。`
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *UnifiedExecuteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的 bash 命令",
			},
			"host": map[string]interface{}{
				"type":        "string",
				"description": "远程主机地址（格式: [user@]host[:port]），不指定则本地执行。默认用户 root，默认端口 22",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "命令执行超时时间（秒），本地默认 30，远程默认 60",
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "命令执行的工作目录（仅本地执行有效）",
			},
			"use_sudo": map[string]interface{}{
				"type":        "boolean",
				"description": "是否使用 sudo 执行命令（仅远程执行有效）",
			},
		},
		"required": []string{"command"},
	}
}

// RiskLevel 返回风险等级
// 无 host → 中风险（动态评估），有 host → 高风险
func (t *UnifiedExecuteTool) RiskLevel() RiskLevel {
	return RiskMedium // 默认中风险，实际风险在 Execute 中动态确定
}

// Execute 执行命令，自动路由本地/远程
func (t *UnifiedExecuteTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	command := parseStringParam(args, "command")
	if command == "" {
		return &Result{
			Success: false,
			Error:   "command 参数不能为空",
		}, fmt.Errorf("command 参数不能为空")
	}

	host := parseStringParam(args, "host")

	if host == "" {
		// 本地执行 — 委托给 LocalBashTool
		result, err := t.localBash.Execute(ctx, args)
		if result != nil && result.Summary != "" {
			result.Summary = "[本地] " + result.Summary
		}
		return result, err
	}

	// 远程执行 — 委托给 SSHExecuteTool
	// 确保 host 参数在 args 中
	remoteArgs := make(map[string]interface{})
	for k, v := range args {
		remoteArgs[k] = v
	}
	remoteArgs["host"] = host

	result, err := t.sshExec.Execute(ctx, remoteArgs)
	// SSHExecuteTool 的 Summary 已包含 [user@host:port]，不需要额外前缀
	return result, err
}
