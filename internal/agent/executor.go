package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"opsxcli/internal/llm"
	"opsxcli/internal/tools"
)

// executeToolCall 执行工具调用
func (a *Agent) executeToolCall(ctx context.Context, toolCall llm.ToolCall, firstCall bool) string {
	// 只在第一次调用时显示工具详情
	if firstCall {
		a.status.ShowToolCall(toolCall.Function.Name)
	}

	// Debug模式: 显示详细信息
	if a.debug {
		a.status.Clear()
		debugColor := color.New(color.FgYellow)
		fmt.Printf("\n%s Tool: %s\n", debugColor.Sprint("[DEBUG]"), toolCall.Function.Name)
		fmt.Printf("%s Args: %s\n", debugColor.Sprint("[DEBUG]"), toolCall.Function.Arguments)
	}

	// 获取工具
	tool, err := a.toolRegistry.Get(toolCall.Function.Name)
	if err != nil {
		errorColor := color.New(color.FgRed)
		a.status.Clear()
		fmt.Printf("  %s %s\n", color.HiBlackString("⎿"), errorColor.Sprintf("错误: %v", err))
		return fmt.Sprintf("错误: %v", err)
	}

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return fmt.Sprintf("参数解析失败: %v", err)
	}

	// 用于记录建议信息
	var hasSuggestion bool
	var suggestionText string

	// opsxcli 工具调用规范验证 - 新策略:引导优先级,不拦截
	validator := tools.NewToolCallValidator()
	isValid, errMsg, suggestion := validator.ValidateToolCall(toolCall.Function.Name, args)

	// 新策略: 不再拦截和自动修正,只记录建议
	if suggestion != "" {
		hasSuggestion = true
		suggestionText = suggestion

		// 在控制台显示建议(不拦截执行)
		if a.debug {
			infoColor := color.New(color.FgCyan)
			a.status.Clear()
			fmt.Printf("  %s 💡 %s\n", color.HiBlackString("⎿"), infoColor.Sprint(suggestion))
		}
	}

	// 旧的拦截逻辑已移除,现在所有调用都允许通过
	_ = isValid // 避免未使用变量警告
	_ = errMsg  // 避免未使用变量警告

	// 安全检查
	approved, auditLog, err := a.safetyController.CheckToolExecution(tool, args)
	if err != nil {
		return fmt.Sprintf("安全检查失败: %v", err)
	}

	if !approved {
		if a.debug {
			debugColor := color.New(color.FgYellow)
			fmt.Printf("%s Tool rejected by user\n\n", debugColor.Sprint("[DEBUG]"))
		}
		return "用户拒绝执行此操作"
	}

	// 执行工具
	toolCtx, cancel := context.WithTimeout(ctx, a.config.ToolTimeout)
	defer cancel()

	result, err := tool.Execute(toolCtx, args)

	// 处理执行错误或失败结果
	if err != nil || (result != nil && !result.Success) {
		errorOutput := ""
		if err != nil {
			errorOutput = err.Error()
		} else if result != nil {
			errorOutput = result.Error + "\n" + result.Output
		}

		// 判断是否应该触发自我修复(智能过滤运维检查命令)
		if !a.shouldTriggerSelfHealing(toolCall.Function.Name, args, result, errorOutput) {
			// 这是正常的检查命令失败,不触发自我修复,直接返回结果
			errMsg := fmt.Sprintf("工具执行异常: %v", err)
			if result != nil {
				errMsg = fmt.Sprintf("执行失败: %s\n输出: %s", result.Error, result.Output)
			}

			a.safetyController.MarkExecuted(auditLog, &tools.ToolResult{
				Success: false,
				Error:   errMsg,
			})
			return errMsg
		}

		// 准备自我修复上下文
		executionContext := map[string]interface{}{
			"command":   fmt.Sprintf("%v", args),
			"tool_name": toolCall.Function.Name,
		}

		// 显示错误检测状态
		a.status.Clear()
		warnColor := color.New(color.FgYellow)
		fmt.Printf("  %s %s\n", color.HiBlackString("⎿"), warnColor.Sprint("检测到错误,尝试自动修复..."))

		// 尝试自我修复
		recovery, recErr := a.selfHealing.Recover(ctx, errorOutput, executionContext)

		if recErr == nil && recovery != nil {
			// 显示恢复建议
			a.status.Clear()
			infoColor := color.New(color.FgCyan)
			fmt.Printf("  %s %s\n", color.HiBlackString("⎿"), infoColor.Sprintf("恢复建议: %s", recovery.Description))
			fmt.Printf("  %s %s\n", color.HiBlackString("⎿"), color.HiBlackString(recovery.Reason))

			if recovery.Success {
				// 自动重试(如果恢复动作标记为可自动执行)
				if recovery.Type == "adjust_params" || recovery.Type == "retry" {
					fmt.Printf("  %s %s\n", color.HiBlackString("⎿"), color.GreenString("自动应用修复并重试..."))

					// 使用恢复后的命令重试
					if recovery.Command != "" {
						// 更新参数中的命令
						if cmdArg, ok := args["command"].(string); ok && cmdArg != "" {
							args["command"] = recovery.Command
						}

						// 重新执行
						retryCtx, retryCancel := context.WithTimeout(ctx, a.config.ToolTimeout)
						defer retryCancel()

						retryResult, retryErr := tool.Execute(retryCtx, args)
						if retryErr == nil && retryResult.Success {
							// 记录成功的恢复动作(供学习)
							a.selfHealing.Learn(errorOutput, recovery)

							a.safetyController.MarkExecuted(auditLog, retryResult)
							fmt.Printf("  %s %s\n\n", color.HiBlackString("⎿"), color.GreenString("✓ 修复成功"))

							return retryResult.Output
						}
					}
				}

				// 如果不是自动重试类型,返回恢复建议
				recoveryMsg := fmt.Sprintf("原始错误: %s\n\n自动修复建议:\n- 类型: %s\n- 描述: %s\n- 原因: %s",
					errorOutput, recovery.Type, recovery.Description, recovery.Reason)

				if recovery.Command != "" {
					recoveryMsg += fmt.Sprintf("\n- 建议命令: %s", recovery.Command)
				}

				// 记录带恢复建议的失败
				a.safetyController.MarkExecuted(auditLog, &tools.ToolResult{
					Success: false,
					Error:   recoveryMsg,
					Output:  errorOutput,
				})

				return recoveryMsg
			}
		}

		// 如果自我修复失败或不可用,返回原始错误
		errMsg := fmt.Sprintf("工具执行异常: %v", err)
		if result != nil {
			errMsg = fmt.Sprintf("执行失败: %s\n输出: %s", result.Error, result.Output)
		}

		a.safetyController.MarkExecuted(auditLog, &tools.ToolResult{
			Success: false,
			Error:   errMsg,
		})
		return errMsg
	}

	// Debug模式: 显示结果
	if a.debug {
		debugColor := color.New(color.FgYellow)
		resultPreview := result.Output
		if len(resultPreview) > 200 {
			resultPreview = resultPreview[:200] + "..."
		}
		fmt.Printf("%s Result: %s\n\n", debugColor.Sprint("[DEBUG]"), resultPreview)
	}

	// 记录执行结果
	a.safetyController.MarkExecuted(auditLog, result)

	// 如果有建议,在返回结果中添加提示(让 LLM 学习)
	if hasSuggestion {
		tipMsg := fmt.Sprintf(
			"💡 提示: %s\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"\n"+
				"执行结果:\n%s",
			suggestionText,
			result.Output,
		)
		return tipMsg
	}

	return result.Output
}

// convertToolsToLLMFormat 转换工具为LLM格式
func (a *Agent) convertToolsToLLMFormat() []llm.Tool {
	toolsList := a.toolRegistry.List()
	llmTools := make([]llm.Tool, len(toolsList))

	for i, tool := range toolsList {
		llmTools[i] = llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
		}
	}

	return llmTools
}

// shouldTriggerSelfHealing 判断是否应该触发自我修复
// 智能过滤运维检查命令,避免对正常的探测行为触发修复
func (a *Agent) shouldTriggerSelfHealing(toolName string, args map[string]interface{}, result *tools.ToolResult, errorOutput string) bool {
	// 非bash命令,需要触发修复
	if toolName != "bash" {
		// 但wget/curl工具如果只是探测,不触发
		if toolName == "wget" || toolName == "curl" {
			// 如果是探测性下载(只检查是否可达),不触发修复
			if url, ok := args["url"].(string); ok {
				if strings.Contains(url, "--spider") || strings.Contains(errorOutput, "404") {
					return false // 404是正常的探测结果
				}
			}
		}
		return true
	}

	// 获取bash命令
	command, ok := args["command"].(string)
	if !ok {
		return true
	}

	// 定义运维检查型命令列表
	probeCommands := []string{
		// 查找和搜索命令
		"grep", "egrep", "fgrep",
		"find",
		"which",
		"whereis",
		"locate",

		// 查看命令 (只读)
		"cat", "head", "tail", "less", "more",
		"ls", "ll", "dir",

		// 进程和网络检查
		"ps", "pgrep",
		"netstat", "ss", "lsof",
		"ping", "telnet", "nc",

		// 测试命令
		"test", "[", "[[",

		// 包管理查询
		"rpm -qa", "rpm -q", "dpkg -l", "dpkg -s",
		"yum list", "dnf list", "apt list",
		"which", "command -v",
	}

	// 检查是否是探测型命令
	commandLower := strings.ToLower(command)
	for _, probe := range probeCommands {
		if strings.Contains(commandLower, probe) {
			// grep没有匹配结果是正常的
			if strings.Contains(commandLower, "grep") {
				return false // 不触发修复
			}

			// which/whereis找不到命令是正常的
			if strings.Contains(commandLower, "which") || strings.Contains(commandLower, "whereis") {
				return false
			}

			// find没有找到文件是正常的
			if strings.Contains(commandLower, "find") {
				return false
			}

			// ls/cat等查看命令,如果文件不存在,也是正常的探测结果
			if strings.Contains(commandLower, "ls") ||
			   strings.Contains(commandLower, "cat") ||
			   strings.Contains(commandLower, "head") ||
			   strings.Contains(commandLower, "tail") {
				// 如果是"no such file"错误,这是正常的探测
				if strings.Contains(errorOutput, "No such file") ||
				   strings.Contains(errorOutput, "cannot access") {
					return false
				}
			}

			// netstat/ss/lsof没有找到端口是正常的
			if strings.Contains(commandLower, "netstat") ||
			   strings.Contains(commandLower, "ss") ||
			   strings.Contains(commandLower, "lsof") {
				return false
			}

			// rpm/yum/dnf/apt查询没有结果是正常的
			if (strings.Contains(commandLower, "rpm") ||
			    strings.Contains(commandLower, "yum") ||
			    strings.Contains(commandLower, "dnf") ||
			    strings.Contains(commandLower, "apt")) &&
			   (strings.Contains(commandLower, "list") ||
			    strings.Contains(commandLower, "-qa") ||
			    strings.Contains(commandLower, "-q")) {
				return false
			}
		}
	}

	// 特殊情况: curl/wget的测试性请求
	if strings.Contains(commandLower, "curl") {
		// 如果包含 -s (silent) 或测试性参数,不触发修复
		if strings.Contains(commandLower, "curl -s") ||
		   strings.Contains(commandLower, "--spider") {
			return false
		}
	}

	// 操作型命令才需要触发修复
	// 例如: systemctl, service, install, yum install, apt-get install, 等
	operationCommands := []string{
		"systemctl start", "systemctl stop", "systemctl restart",
		"service start", "service stop", "service restart",
		"yum install", "dnf install", "apt-get install", "apt install",
		"docker run", "docker start", "docker exec",
		"kubectl apply", "kubectl create",
		"nohup", "sh ", "bash ",
	}

	for _, op := range operationCommands {
		if strings.Contains(commandLower, op) {
			return true // 这些操作失败需要修复
		}
	}

	// 默认: 如果是明确的操作命令,触发修复
	// 如果不确定,不触发修复(避免误报)
	return false
}
