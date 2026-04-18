package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"opsxcli/internal/llm"
)

// ErrorPattern 错误模式
type ErrorPattern struct {
	Name        string                // 错误名称
	Pattern     *regexp.Regexp        // 错误匹配模式
	Category    string                // 错误分类
	Severity    string                // 严重程度
	Handler     ErrorHandler          // 错误处理器
	Description string                // 错误描述
}

// ErrorHandler 错误处理器函数
type ErrorHandler func(ctx context.Context, error string, context map[string]interface{}) (*RecoveryAction, error)

// RecoveryAction 恢复动作
type RecoveryAction struct {
	Type        string                 // 动作类型: retry, adjust_params, alternative_approach
	Description string                 // 动作描述
	Parameters  map[string]interface{} // 调整后的参数
	Command     string                 // 替代命令
	Reason      string                 // 原因说明
	Success     bool                   // 是否成功
}

// SelfHealingSystem 自我修正系统
type SelfHealingSystem struct {
	patterns    []ErrorPattern
	maxRetries  int
	llmClient   llm.Client
	learningDB  map[string]*RecoveryAction // 学习数据库
}

// NewSelfHealingSystem 创建自我修正系统
func NewSelfHealingSystem(llmClient llm.Client) *SelfHealingSystem {
	sys := &SelfHealingSystem{
		patterns:   make([]ErrorPattern, 0),
		maxRetries: 3,
		llmClient:  llmClient,
		learningDB: make(map[string]*RecoveryAction),
	}

	// 注册内置错误模式
	sys.registerBuiltinPatterns()

	return sys
}

// registerBuiltinPatterns 注册内置错误模式
func (s *SelfHealingSystem) registerBuiltinPatterns() {
	// 1. 内存不足错误
	s.RegisterPattern(ErrorPattern{
		Name:        "OutOfMemory",
		Pattern:     regexp.MustCompile(`(?i)(cannot allocate memory|out of memory|insufficient memory|oom)`),
		Category:    "resource",
		Severity:    "high",
		Handler:     s.handleOutOfMemory,
		Description: "系统内存不足",
	})

	// 2. 权限拒绝错误
	s.RegisterPattern(ErrorPattern{
		Name:        "PermissionDenied",
		Pattern:     regexp.MustCompile(`(?i)(permission denied|access denied|operation not permitted)`),
		Category:    "permission",
		Severity:    "high",
		Handler:     s.handlePermissionDenied,
		Description: "权限不足",
	})

	// 3. 文件不存在错误
	s.RegisterPattern(ErrorPattern{
		Name:        "FileNotFound",
		Pattern:     regexp.MustCompile(`(?i)(no such file or directory|file not found|cannot find)`),
		Category:    "filesystem",
		Severity:    "medium",
		Handler:     s.handleFileNotFound,
		Description: "文件或目录不存在",
	})

	// 4. 端口已占用错误
	s.RegisterPattern(ErrorPattern{
		Name:        "PortInUse",
		Pattern:     regexp.MustCompile(`(?i)(address already in use|port.*already.*use|bind.*failed)`),
		Category:    "network",
		Severity:    "medium",
		Handler:     s.handlePortInUse,
		Description: "端口已被占用",
	})

	// 5. 磁盘空间不足错误
	s.RegisterPattern(ErrorPattern{
		Name:        "DiskFull",
		Pattern:     regexp.MustCompile(`(?i)(no space left|disk.*full|quota exceeded)`),
		Category:    "resource",
		Severity:    "high",
		Handler:     s.handleDiskFull,
		Description: "磁盘空间不足",
	})

	// 6. 命令不存在错误
	s.RegisterPattern(ErrorPattern{
		Name:        "CommandNotFound",
		Pattern:     regexp.MustCompile(`(?i)(command not found|not found in PATH)`),
		Category:    "environment",
		Severity:    "high",
		Handler:     s.handleCommandNotFound,
		Description: "命令未找到",
	})

	// 7. 依赖缺失错误
	s.RegisterPattern(ErrorPattern{
		Name:        "DependencyMissing",
		Pattern:     regexp.MustCompile(`(?i)(cannot find package|module not found|dependency.*missing)`),
		Category:    "dependency",
		Severity:    "high",
		Handler:     s.handleDependencyMissing,
		Description: "依赖包缺失",
	})

	// 8. 超时错误
	s.RegisterPattern(ErrorPattern{
		Name:        "Timeout",
		Pattern:     regexp.MustCompile(`(?i)(timeout|timed out|connection refused)`),
		Category:    "network",
		Severity:    "medium",
		Handler:     s.handleTimeout,
		Description: "连接超时",
	})

	// 9. 进程已存在错误
	s.RegisterPattern(ErrorPattern{
		Name:        "ProcessExists",
		Pattern:     regexp.MustCompile(`(?i)(already running|process.*exists|lock file.*exists)`),
		Category:    "process",
		Severity:    "low",
		Handler:     s.handleProcessExists,
		Description: "进程已在运行",
	})
}

// RegisterPattern 注册错误模式
func (s *SelfHealingSystem) RegisterPattern(pattern ErrorPattern) {
	s.patterns = append(s.patterns, pattern)
}

// DetectError 检测错误
func (s *SelfHealingSystem) DetectError(output string) (*ErrorPattern, bool) {
	for _, pattern := range s.patterns {
		if pattern.Pattern.MatchString(output) {
			return &pattern, true
		}
	}
	return nil, false
}

// Recover 执行恢复操作
func (s *SelfHealingSystem) Recover(ctx context.Context, errorOutput string, executionContext map[string]interface{}) (*RecoveryAction, error) {
	// 检测错误类型
	pattern, found := s.DetectError(errorOutput)
	if !found {
		// 使用 LLM 进行智能分析
		return s.intelligentRecover(ctx, errorOutput, executionContext)
	}

	// 使用对应的错误处理器
	return pattern.Handler(ctx, errorOutput, executionContext)
}

// handleOutOfMemory 处理内存不足错误
func (s *SelfHealingSystem) handleOutOfMemory(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	// 提取 JVM 内存参数
	xmsPattern := regexp.MustCompile(`-Xms(\d+)([gGmM])`)
	xmxPattern := regexp.MustCompile(`-Xmx(\d+)([gGmM])`)

	// 获取系统可用内存
	command, ok := context["command"].(string)
	if !ok {
		return nil, fmt.Errorf("无法获取命令上下文")
	}

	// 分析内存需求
	var newXms, newXmx string
	if strings.Contains(command, "Xms") {
		// 检测到 JVM 参数,降低内存需求
		xmsMatches := xmsPattern.FindStringSubmatch(command)
		xmxMatches := xmxPattern.FindStringSubmatch(command)

		if len(xmsMatches) > 0 && len(xmxMatches) > 0 {
			// 将内存需求降低到 512M
			newXms = "-Xms512m"
			newXmx = "-Xmx512m"

			// 替换命令中的内存参数
			newCommand := xmsPattern.ReplaceAllString(command, newXms)
			newCommand = xmxPattern.ReplaceAllString(newCommand, newXmx)

			return &RecoveryAction{
				Type:        "adjust_params",
				Description: "检测到内存不足,自动降低 JVM 内存参数",
				Parameters: map[string]interface{}{
					"original_xms": xmsMatches[0],
					"original_xmx": xmxMatches[0],
					"new_xms":      newXms,
					"new_xmx":      newXmx,
				},
				Command: newCommand,
				Reason:  "系统可用内存不足以支持原始配置,自动调整为 512MB",
				Success: true,
			}, nil
		}
	}

	// 如果不是 JVM 相关,建议释放内存
	return &RecoveryAction{
		Type:        "alternative_approach",
		Description: "内存不足,建议释放系统资源或增加 swap 空间",
		Command:     "echo 3 > /proc/sys/vm/drop_caches && swapoff -a && swapon -a",
		Reason:      "系统内存不足,尝试释放缓存",
		Success:     false, // 需要用户确认
	}, nil
}

// handlePermissionDenied 处理权限拒绝错误
func (s *SelfHealingSystem) handlePermissionDenied(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	command, ok := context["command"].(string)
	if !ok {
		return nil, fmt.Errorf("无法获取命令上下文")
	}

	// 如果命令不以 sudo 开头,添加 sudo
	if !strings.HasPrefix(strings.TrimSpace(command), "sudo") {
		return &RecoveryAction{
			Type:        "adjust_params",
			Description: "检测到权限不足,使用 sudo 重试",
			Command:     "sudo " + command,
			Reason:      "当前用户权限不足,需要管理员权限",
			Success:     true,
		}, nil
	}

	return &RecoveryAction{
		Type:        "alternative_approach",
		Description: "权限不足且已使用 sudo,可能需要修改文件权限或 SELinux 设置",
		Reason:      "即使使用 sudo 仍然权限不足,可能是文件系统权限或 SELinux 限制",
		Success:     false,
	}, nil
}

// handleFileNotFound 处理文件不存在错误
func (s *SelfHealingSystem) handleFileNotFound(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	// 提取文件路径
	pathPattern := regexp.MustCompile(`['"]?(/[^'":\s]+)['"]?`)
	matches := pathPattern.FindStringSubmatch(errorOutput)

	if len(matches) > 1 {
		missingPath := matches[1]

		// 尝试在常见位置查找文件
		return &RecoveryAction{
			Type:        "alternative_approach",
			Description: fmt.Sprintf("文件 %s 不存在,尝试在其他位置查找", missingPath),
			Command:     fmt.Sprintf("find /opt /usr/local /home -name '%s' 2>/dev/null | head -5", missingPath),
			Reason:      "文件路径可能不正确,尝试搜索正确位置",
			Success:     false,
		}, nil
	}

	return nil, fmt.Errorf("无法提取缺失文件路径")
}

// handlePortInUse 处理端口占用错误
func (s *SelfHealingSystem) handlePortInUse(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	// 提取端口号
	portPattern := regexp.MustCompile(`port[:\s]+(\d+)|:(\d+)`)
	matches := portPattern.FindStringSubmatch(errorOutput)

	var port string
	if len(matches) > 1 {
		if matches[1] != "" {
			port = matches[1]
		} else if matches[2] != "" {
			port = matches[2]
		}
	}

	if port != "" {
		return &RecoveryAction{
			Type:        "alternative_approach",
			Description: fmt.Sprintf("端口 %s 已被占用,查找并停止占用进程", port),
			Command:     fmt.Sprintf("lsof -ti:%s | xargs kill -9", port),
			Reason:      fmt.Sprintf("端口 %s 被其他进程占用,尝试释放端口", port),
			Success:     false, // 需要用户确认
		}, nil
	}

	return &RecoveryAction{
		Type:        "alternative_approach",
		Description: "端口已被占用,建议使用其他端口",
		Reason:      "无法确定具体端口号,建议手动检查",
		Success:     false,
	}, nil
}

// handleDiskFull 处理磁盘空间不足错误
func (s *SelfHealingSystem) handleDiskFull(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	return &RecoveryAction{
		Type:        "alternative_approach",
		Description: "磁盘空间不足,建议清理临时文件和日志",
		Command:     "du -sh /var/log/* /tmp/* | sort -rh | head -10",
		Reason:      "磁盘空间不足,需要清理空间",
		Success:     false,
	}, nil
}

// handleCommandNotFound 处理命令不存在错误
func (s *SelfHealingSystem) handleCommandNotFound(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	// 提取命令名称
	cmdPattern := regexp.MustCompile(`command\s+['"]?(\w+)['"]?\s+not found`)
	matches := cmdPattern.FindStringSubmatch(errorOutput)

	if len(matches) > 1 {
		missingCmd := matches[1]

		return &RecoveryAction{
			Type:        "alternative_approach",
			Description: fmt.Sprintf("命令 %s 未找到,尝试安装", missingCmd),
			Command:     fmt.Sprintf("yum install -y %s || apt-get install -y %s", missingCmd, missingCmd),
			Reason:      fmt.Sprintf("命令 %s 未安装,尝试自动安装", missingCmd),
			Success:     false,
		}, nil
	}

	return nil, fmt.Errorf("无法提取缺失命令名称")
}

// handleDependencyMissing 处理依赖缺失错误
func (s *SelfHealingSystem) handleDependencyMissing(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	return &RecoveryAction{
		Type:        "alternative_approach",
		Description: "依赖包缺失,建议使用包管理器安装",
		Reason:      "检测到依赖包缺失",
		Success:     false,
	}, nil
}

// handleTimeout 处理超时错误
func (s *SelfHealingSystem) handleTimeout(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	command, ok := context["command"].(string)
	if !ok {
		return nil, fmt.Errorf("无法获取命令上下文")
	}

	return &RecoveryAction{
		Type:        "retry",
		Description: "连接超时,增加超时时间并重试",
		Command:     command,
		Parameters: map[string]interface{}{
			"timeout": 60, // 增加到 60 秒
		},
		Reason:  "网络连接超时,使用更长的超时时间重试",
		Success: true,
	}, nil
}

// handleProcessExists 处理进程已存在错误
func (s *SelfHealingSystem) handleProcessExists(ctx context.Context, errorOutput string, context map[string]interface{}) (*RecoveryAction, error) {
	return &RecoveryAction{
		Type:        "alternative_approach",
		Description: "进程已在运行,跳过启动操作",
		Reason:      "检测到进程已存在,无需重复启动",
		Success:     true,
	}, nil
}

// intelligentRecover 使用 LLM 进行智能恢复分析
func (s *SelfHealingSystem) intelligentRecover(ctx context.Context, errorOutput string, executionContext map[string]interface{}) (*RecoveryAction, error) {
	// 构建分析提示
	prompt := fmt.Sprintf(`分析以下错误输出并提供恢复建议:

错误输出:
%s

执行上下文:
命令: %v

请分析:
1. 错误的根本原因
2. 可能的解决方案
3. 具体的恢复命令或参数调整

以 JSON 格式返回:
{
  "error_type": "错误类型",
  "root_cause": "根本原因",
  "recovery_command": "恢复命令",
  "reason": "原因说明"
}`, errorOutput, executionContext["command"])

	// 调用 LLM 分析
	resp, err := s.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "你是一个专业的系统运维专家,擅长分析错误并提供解决方案。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   1024,
	})

	if err != nil {
		return nil, fmt.Errorf("LLM 分析失败: %w", err)
	}

	// 解析响应
	return &RecoveryAction{
		Type:        "intelligent_analysis",
		Description: "基于 AI 分析的恢复建议",
		Command:     resp.Message.Content,
		Reason:      "使用 LLM 进行智能错误分析",
		Success:     false, // 需要用户确认
	}, nil
}

// Learn 学习成功的恢复动作
func (s *SelfHealingSystem) Learn(errorPattern string, action *RecoveryAction) {
	if action.Success {
		s.learningDB[errorPattern] = action
	}
}
