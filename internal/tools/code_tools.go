package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileReadTool 读取文件工具
type FileReadTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	riskLevel   RiskLevel
}

func NewFileReadTool() *FileReadTool {
	return &FileReadTool{
		name:        "file_read",
		description: "读取文件内容。可以指定行数范围。适用于查看代码、配置文件等。",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径（相对或绝对路径）",
				},
				"start_line": map[string]interface{}{
					"type":        "integer",
					"description": "起始行号（可选，从1开始）",
				},
				"end_line": map[string]interface{}{
					"type":        "integer",
					"description": "结束行号（可选）",
				},
			},
			"required": []string{"path"},
		},
		riskLevel: RiskSafe,
	}
}

func (t *FileReadTool) Name() string                                { return t.name }
func (t *FileReadTool) Description() string                         { return t.description }
func (t *FileReadTool) Parameters() map[string]interface{}          { return t.parameters }
func (t *FileReadTool) RiskLevel() RiskLevel                        { return t.riskLevel }

func (t *FileReadTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path := parseStringParam(args, "path")
	if path == "" {
		return &ToolResult{Success: false, Error: "path 参数必须是字符串"}, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效的路径: %v", err)}, nil
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("读取文件失败: %v", err)}, nil
	}

	result := string(content)

	// 处理行数范围
	startLine := parseIntParam(args, "start_line")
	endLine := parseIntParam(args, "end_line")

	if startLine > 0 {
		lines := strings.Split(result, "\n")
		start := startLine - 1
		if start < 0 {
			start = 0
		}

		end := len(lines)
		if endLine > 0 {
			end = endLine
			if end > len(lines) {
				end = len(lines)
			}
		}

		if start < len(lines) {
			result = strings.Join(lines[start:end], "\n")
		}
	}

	return &ToolResult{Success: true, Output: result}, nil
}

// FileWriteTool 写入文件工具
type FileWriteTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	riskLevel   RiskLevel
}

func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{
		name:        "file_write",
		description: "创建或覆盖文件内容。危险操作，会覆盖现有文件！适用于创建新文件或完全替换文件内容。",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文件内容",
				},
			},
			"required": []string{"path", "content"},
		},
		riskLevel: RiskHigh,
	}
}

func (t *FileWriteTool) Name() string                       { return t.name }
func (t *FileWriteTool) Description() string                { return t.description }
func (t *FileWriteTool) Parameters() map[string]interface{} { return t.parameters }
func (t *FileWriteTool) RiskLevel() RiskLevel               { return t.riskLevel }

func (t *FileWriteTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path := parseStringParam(args, "path")
	content := parseStringParam(args, "content")

	if path == "" {
		return &ToolResult{Success: false, Error: "path 参数必须是字符串"}, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效的路径: %v", err)}, nil
	}

	// 创建目录（如果不存在）
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("创建目录失败: %v", err)}, nil
	}

	err = os.WriteFile(absPath, []byte(content), 0644)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("写入文件失败: %v", err)}, nil
	}

	return &ToolResult{Success: true, Output: fmt.Sprintf("已写入文件: %s (%d bytes)", path, len(content))}, nil
}

// FileEditTool 编辑文件工具（查找替换）
type FileEditTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	riskLevel   RiskLevel
}

func NewFileEditTool() *FileEditTool {
	return &FileEditTool{
		name:        "file_edit",
		description: "编辑文件：查找并替换指定内容。用于对现有文件进行局部修改。",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径",
				},
				"search": map[string]interface{}{
					"type":        "string",
					"description": "要查找的文本",
				},
				"replace": map[string]interface{}{
					"type":        "string",
					"description": "替换为的文本",
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "是否替换所有匹配（默认只替换第一个）",
				},
			},
			"required": []string{"path", "search", "replace"},
		},
		riskLevel: RiskHigh,
	}
}

func (t *FileEditTool) Name() string                       { return t.name }
func (t *FileEditTool) Description() string                { return t.description }
func (t *FileEditTool) Parameters() map[string]interface{} { return t.parameters }
func (t *FileEditTool) RiskLevel() RiskLevel               { return t.riskLevel }

func (t *FileEditTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path := parseStringParam(args, "path")
	search := parseStringParam(args, "search")
	replace := parseStringParam(args, "replace")
	replaceAll := parseBoolParam(args, "all")

	absPath, err := filepath.Abs(path)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("无效的路径: %v", err)}, nil
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("读取文件失败: %v", err)}, nil
	}

	original := string(content)
	var result string
	var count int

	if replaceAll {
		result = strings.ReplaceAll(original, search, replace)
		count = strings.Count(original, search)
	} else {
		result = strings.Replace(original, search, replace, 1)
		if strings.Contains(original, search) {
			count = 1
		}
	}

	if count == 0 {
		return &ToolResult{Success: false, Error: fmt.Sprintf("未找到匹配的文本: %s", search)}, nil
	}

	err = os.WriteFile(absPath, []byte(result), 0644)
	if err != nil {
		return &ToolResult{Success: false, Error: fmt.Sprintf("写入文件失败: %v", err)}, nil
	}

	return &ToolResult{Success: true, Output: fmt.Sprintf("已替换 %d 处匹配", count)}, nil
}

// GitStatusTool Git 状态工具
type GitStatusTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	riskLevel   RiskLevel
}

func NewGitStatusTool() *GitStatusTool {
	return &GitStatusTool{
		name:        "git_status",
		description: "查看 Git 仓库状态，显示已修改、已添加、未跟踪的文件",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "仓库路径（默认为当前目录）",
				},
			},
		},
		riskLevel: RiskSafe,
	}
}

func (t *GitStatusTool) Name() string                       { return t.name }
func (t *GitStatusTool) Description() string                { return t.description }
func (t *GitStatusTool) Parameters() map[string]interface{} { return t.parameters }
func (t *GitStatusTool) RiskLevel() RiskLevel               { return t.riskLevel }

func (t *GitStatusTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path := parseStringParam(args, "path")
	if path == "" {
		path = "."
	}

	cmd := exec.CommandContext(ctx, "git", "-C", path, "status", "--short")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Success: false, Output: string(output), Error: fmt.Sprintf("执行 git status 失败: %v", err)}, nil
	}

	return &ToolResult{Success: true, Output: string(output)}, nil
}

// GitDiffTool Git diff 工具
type GitDiffTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	riskLevel   RiskLevel
}

func NewGitDiffTool() *GitDiffTool {
	return &GitDiffTool{
		name:        "git_diff",
		description: "查看 Git 变更差异，显示文件的具体修改内容",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "仓库路径（默认为当前目录）",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "指定文件（可选）",
				},
				"cached": map[string]interface{}{
					"type":        "boolean",
					"description": "查看暂存区变更",
				},
			},
		},
		riskLevel: RiskSafe,
	}
}

func (t *GitDiffTool) Name() string                       { return t.name }
func (t *GitDiffTool) Description() string                { return t.description }
func (t *GitDiffTool) Parameters() map[string]interface{} { return t.parameters }
func (t *GitDiffTool) RiskLevel() RiskLevel               { return t.riskLevel }

func (t *GitDiffTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path := parseStringParam(args, "path")
	if path == "" {
		path = "."
	}

	cmdArgs := []string{"-C", path, "diff"}

	if parseBoolParam(args, "cached") {
		cmdArgs = append(cmdArgs, "--cached")
	}

	file := parseStringParam(args, "file")
	if file != "" {
		cmdArgs = append(cmdArgs, file)
	}

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Success: false, Output: string(output), Error: fmt.Sprintf("执行 git diff 失败: %v", err)}, nil
	}

	return &ToolResult{Success: true, Output: string(output)}, nil
}

// CodeSearchTool 代码搜索工具
type CodeSearchTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	riskLevel   RiskLevel
}

func NewCodeSearchTool() *CodeSearchTool {
	return &CodeSearchTool{
		name:        "code_search",
		description: "在代码库中搜索文本或模式（使用 grep），支持正则表达式",
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "搜索模式（支持正则表达式）",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "搜索路径（默认为当前目录）",
				},
				"file_pattern": map[string]interface{}{
					"type":        "string",
					"description": "文件名模式（如 *.go）",
				},
			},
			"required": []string{"pattern"},
		},
		riskLevel: RiskSafe,
	}
}

func (t *CodeSearchTool) Name() string                       { return t.name }
func (t *CodeSearchTool) Description() string                { return t.description }
func (t *CodeSearchTool) Parameters() map[string]interface{} { return t.parameters }
func (t *CodeSearchTool) RiskLevel() RiskLevel               { return t.riskLevel }

func (t *CodeSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	pattern := parseStringParam(args, "pattern")
	path := parseStringParam(args, "path")
	if path == "" {
		path = "."
	}

	cmdArgs := []string{"-r", "-n", pattern, path}

	filePattern := parseStringParam(args, "file_pattern")
	if filePattern != "" {
		cmdArgs = append(cmdArgs, "--include", filePattern)
	}

	cmd := exec.CommandContext(ctx, "grep", cmdArgs...)
	output, err := cmd.CombinedOutput()

	// grep 返回 1 表示没有匹配，这不是错误
	if err != nil && len(output) == 0 {
		return &ToolResult{Success: true, Output: "未找到匹配结果"}, nil
	}

	return &ToolResult{Success: true, Output: string(output)}, nil
}
