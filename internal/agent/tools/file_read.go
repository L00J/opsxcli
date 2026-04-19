// file_read.go - 文件读取工具
// 支持 offset/limit 分页、行号显示、二进制检测、权限检查
package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// 二进制检测阈值
const binaryCheckSize = 512

// FileReadTool 文件读取工具
type FileReadTool struct{}

// NewFileReadTool 创建文件读取工具
func NewFileReadTool() *FileReadTool {
	return &FileReadTool{}
}

// Name 返回工具名称
func (t *FileReadTool) Name() string {
	return "file_read"
}

// Description 返回工具描述
func (t *FileReadTool) Description() string {
	return "读取文件内容，支持分页、行号显示、二进制检测和权限检查。适合读取文本文件、配置文件、日志文件等"
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *FileReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "要读取的文件路径（绝对路径或相对路径）",
			},
			"offset": map[string]interface{}{
				"type":        "number",
				"description": "起始行号（从 1 开始），默认 1",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "读取的最大行数，默认 500，最大 2000",
			},
		},
		"required": []string{"path"},
	}
}

// RiskLevel 返回风险等级（只读操作，安全）
func (t *FileReadTool) RiskLevel() RiskLevel {
	return RiskSafe
}

// FileLine 文件行结构
type FileLine struct {
	Num     int    `json:"num"`
	Content string `json:"content"`
}

// FileReadResult 文件读取结果结构
type FileReadResult struct {
	Path        string     `json:"path"`
	TotalLines  int        `json:"total_lines"`
	Lines       []FileLine `json:"lines"`
	IsBinary    bool       `json:"is_binary,omitempty"`
	Truncated   bool       `json:"truncated,omitempty"`
	ReadLines   int        `json:"read_lines"`
	StartLine   int        `json:"start_line"`
}

// Execute 执行文件读取
func (t *FileReadTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	// 解析参数
	path := parseStringParam(args, "path")
	if path == "" {
		return &Result{
			Success: false,
			Error:   "path 参数不能为空",
		}, fmt.Errorf("path 参数不能为空")
	}

	// 展开路径（支持 ~ 等）
	path = expandPath(path)

	// 解析 offset（默认 1）
	offset := 1
	if v, ok := parseNumberParam(args, "offset"); ok && v > 0 {
		offset = v
	}

	// 解析 limit（默认 500，最大 2000）
	limit := 500
	if v, ok := parseNumberParam(args, "limit"); ok && v > 0 {
		limit = v
	}
	if limit > 2000 {
		limit = 2000
	}

	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("文件不存在: %s", path),
			}, fmt.Errorf("文件不存在: %s", path)
		}
		if os.IsPermission(err) {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("权限不足，无法访问: %s", path),
			}, fmt.Errorf("权限不足: %s", path)
		}
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("无法访问文件: %s, 错误: %v", path, err),
		}, fmt.Errorf("无法访问文件: %s, %w", path, err)
	}

	// 检查是否是目录
	if info.IsDir() {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("路径是目录而非文件: %s", path),
		}, fmt.Errorf("路径是目录: %s", path)
	}

	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		if os.IsPermission(err) {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("权限不足，无法读取: %s", path),
			}, fmt.Errorf("权限不足: %s", path)
		}
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("打开文件失败: %s, 错误: %v", path, err),
		}, fmt.Errorf("打开文件失败: %s, %w", path, err)
	}
	defer file.Close()

	// 检测二进制文件
	isBinary, err := detectBinaryFile(file)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("检测文件类型失败: %v", err),
		}, fmt.Errorf("检测文件类型失败: %w", err)
	}

	if isBinary {
		return &Result{
			Success: true,
			Output:  fmt.Sprintf("文件 %s 是二进制文件，无法以文本方式读取（大小: %s）", path, formatFileSize(info.Size())),
			Summary: fmt.Sprintf("二进制文件，大小 %s", formatFileSize(info.Size())),
		}, nil
	}

	// 重置文件指针
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("重置文件指针失败: %v", err),
		}, fmt.Errorf("重置文件指针失败: %w", err)
	}

	// 读取文件内容（支持分页）
	lines, totalLines, truncated, err := readFileLines(file, offset, limit)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("读取文件失败: %v", err),
		}, fmt.Errorf("读取文件失败: %w", err)
	}

	// 构建结果
	readResult := FileReadResult{
		Path:       path,
		TotalLines: totalLines,
		Lines:      lines,
		Truncated:  truncated,
		ReadLines:  len(lines),
		StartLine:  offset,
	}

	// 序列化为 JSON
	jsonOutput, err := json.MarshalIndent(readResult, "", "  ")
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("序列化结果失败: %v", err),
		}, fmt.Errorf("序列化结果失败: %w", err)
	}

	summary := fmt.Sprintf("读取文件 %s，共 %d 行，当前显示第 %d-%d 行",
		filepath.Base(path), totalLines, offset, offset+len(lines)-1)
	if truncated {
		summary += "（结果已截断）"
	}

	return &Result{
		Success: true,
		Output:  string(jsonOutput),
		Summary: summary,
	}, nil
}

// detectBinaryFile 检测文件是否为二进制文件
// 读取前 512 字节检查是否包含空字节
func detectBinaryFile(file *os.File) (bool, error) {
	buf := make([]byte, binaryCheckSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	buf = buf[:n]

	// 检查空字节（二进制文件的强特征）
	for _, b := range buf {
		if b == 0 {
			return true, nil
		}
	}

	return false, nil
}

// readFileLines 读取文件行（支持 offset/limit 分页）
func readFileLines(file io.Reader, offset, limit int) ([]FileLine, int, bool, error) {
	scanner := bufio.NewScanner(file)
	// 增大缓冲区以支持长行
	scanner.Buffer(make([]byte, 0), 1024*1024) // 1MB 最大行长度

	var lines []FileLine
	lineNum := 0
	totalLines := 0
	truncated := false

	for scanner.Scan() {
		lineNum++
		totalLines++

		// 跳过 offset 之前的行
		if lineNum < offset {
			continue
		}

		// 检查是否超过 limit
		if len(lines) >= limit {
			// 继续计数总行数，但不存储内容
			truncated = true
			continue
		}

		lines = append(lines, FileLine{
			Num:     lineNum,
			Content: scanner.Text(),
		})
	}

	// 如果截断了，需要知道真正的总行数
	if truncated {
		for scanner.Scan() {
			totalLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, false, err
	}

	return lines, totalLines, truncated, nil
}

// expandPath 展开文件路径（支持 ~ 和环境变量）
func expandPath(path string) string {
	// 展开家目录
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	// 展开环境变量
	path = os.ExpandEnv(path)
	// 获取绝对路径
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
