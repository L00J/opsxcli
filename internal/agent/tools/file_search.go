// file_search.go - 文件搜索工具
// 类似 grep -rn 但返回结构化 JSON，支持内容搜索和文件名搜索
package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// maxSearchResults 最大搜索结果数
const maxSearchResults = 50

// FileSearchTool 文件搜索工具
type FileSearchTool struct{}

// NewFileSearchTool 创建文件搜索工具
func NewFileSearchTool() *FileSearchTool {
	return &FileSearchTool{}
}

// Name 返回工具名称
func (t *FileSearchTool) Name() string {
	return "file_search"
}

// Description 返回工具描述
func (t *FileSearchTool) Description() string {
	return "搜索文件内容或文件名。支持正则表达式匹配，返回结构化 JSON 结果。类似 grep -rn 搜索内容或 find 搜索文件名"
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *FileSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "搜索模式（支持正则表达式）",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "搜索的目录或文件路径，默认当前目录",
			},
			"target": map[string]interface{}{
				"type":        "string",
				"description": "搜索目标类型: content(搜索文件内容) 或 files(按文件名搜索)",
				"enum":        []string{"content", "files"},
			},
			"file_glob": map[string]interface{}{
				"type":        "string",
				"description": "文件名过滤模式（如 *.go, *.py），仅搜索匹配的文件",
			},
		},
		"required": []string{"pattern"},
	}
}

// RiskLevel 返回风险等级（只读操作，安全）
func (t *FileSearchTool) RiskLevel() RiskLevel {
	return RiskSafe
}

// SearchMatch 搜索匹配结果
type SearchMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Content string `json:"content,omitempty"`
}

// FileSearchResult 搜索结果
type FileSearchResult struct {
	Pattern  string         `json:"pattern"`
	Path     string         `json:"path"`
	Target   string         `json:"target"`
	Matches  []SearchMatch  `json:"matches"`
	Total    int            `json:"total"`
	Truncated bool          `json:"truncated,omitempty"`
	FileGlob string         `json:"file_glob,omitempty"`
}

// Execute 执行文件搜索
func (t *FileSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	// 解析参数
	pattern := parseStringParam(args, "pattern")
	if pattern == "" {
		return &Result{
			Success: false,
			Error:   "pattern 参数不能为空",
		}, fmt.Errorf("pattern 参数不能为空")
	}

	// 解析路径（默认当前目录）
	searchPath := parseStringParam(args, "path")
	if searchPath == "" {
		searchPath = "."
	}
	searchPath = expandPath(searchPath)

	// 解析搜索目标（默认 content）
	target := parseStringParam(args, "target")
	if target == "" {
		target = "content"
	}
	if target != "content" && target != "files" {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("不支持的 target 类型: %s，支持 content 或 files", target),
		}, fmt.Errorf("不支持的 target: %s", target)
	}

	// 解析文件名过滤
	fileGlob := parseStringParam(args, "file_glob")

	// 编译正则表达式
	re, err := regexp.Compile(pattern)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("正则表达式编译失败: %v", err),
		}, fmt.Errorf("正则表达式编译失败: %w", err)
	}

	// 检查搜索路径是否存在
	info, err := os.Stat(searchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Result{
				Success: false,
				Error:   fmt.Sprintf("搜索路径不存在: %s", searchPath),
			}, fmt.Errorf("搜索路径不存在: %s", searchPath)
		}
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("无法访问路径: %s, %v", searchPath, err),
		}, fmt.Errorf("无法访问路径: %s, %w", searchPath, err)
	}

	var matches []SearchMatch
	var truncated bool

	if target == "files" {
		// 按文件名搜索
		matches, truncated, err = searchFileNames(re, searchPath, info, fileGlob)
	} else {
		// 按文件内容搜索
		matches, truncated, err = searchFileContent(ctx, re, searchPath, info, fileGlob)
	}

	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("搜索失败: %v", err),
		}, fmt.Errorf("搜索失败: %w", err)
	}

	// 构建结果
	result := FileSearchResult{
		Pattern:   pattern,
		Path:      searchPath,
		Target:    target,
		Matches:   matches,
		Total:     len(matches),
		Truncated: truncated,
		FileGlob:  fileGlob,
	}

	// 序列化为 JSON
	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("序列化结果失败: %v", err),
		}, fmt.Errorf("序列化结果失败: %w", err)
	}

	summary := fmt.Sprintf("搜索完成: 模式 '%s' 在 %s 中找到 %d 个匹配", pattern, searchPath, len(matches))
	if truncated {
		summary += fmt.Sprintf("（仅显示前 %d 条）", maxSearchResults)
	}

	return &Result{
		Success: true,
		Output:  string(jsonOutput),
		Summary: summary,
	}, nil
}

// searchFileContent 搜索文件内容
func searchFileContent(ctx context.Context, re *regexp.Regexp, searchPath string, info fs.FileInfo, fileGlob string) ([]SearchMatch, bool, error) {
	var matches []SearchMatch
	truncated := false

	// 定义处理单个文件的函数
	processFile := func(filePath string) error {
		// 检查 context 取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 检查文件名过滤
		if fileGlob != "" {
			matched, err := filepath.Match(fileGlob, filepath.Base(filePath))
			if err != nil || !matched {
				return nil
			}
		}

		// 跳过二进制文件和隐藏文件
		if shouldSkipFile(filePath) {
			return nil
		}

		file, err := os.Open(filePath)
		if err != nil {
			return nil // 跳过无法打开的文件
		}
		defer file.Close()

		// 检测二进制文件（快速检查前 512 字节）
		if isBinaryFile(file) {
			return nil
		}

		// 重置文件指针
		if _, err := file.Seek(0, 0); err != nil {
			return nil
		}

		// 逐行扫描
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0), 1024*1024) // 1MB 最大行长度
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			if re.MatchString(line) {
				// 截断过长的行
				content := line
				if len(content) > 500 {
					content = content[:500] + "..."
				}

				matches = append(matches, SearchMatch{
					File:    filePath,
					Line:    lineNum,
					Content: strings.TrimSpace(content),
				})

				if len(matches) >= maxSearchResults {
					truncated = true
					return nil
				}
			}
		}
		return scanner.Err()
	}

	// 根据路径类型处理
	if !info.IsDir() {
		_ = processFile(searchPath)
	} else {
		// 遍历目录
		err := filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // 跳过错误
			}

			// 检查 context 取消
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// 跳过隐藏目录和常见的跳过目录
			if d.IsDir() {
				if shouldSkipDir(path) {
					return fs.SkipDir
				}
				return nil
			}

			if truncated {
				return fs.SkipAll
			}

			return processFile(path)
		})
		if err != nil && err != context.Canceled && err != fs.SkipAll {
			return nil, false, err
		}
	}

	return matches, truncated, nil
}

// searchFileNames 按文件名搜索
func searchFileNames(re *regexp.Regexp, searchPath string, info fs.FileInfo, fileGlob string) ([]SearchMatch, bool, error) {
	var matches []SearchMatch
	truncated := false

	// 根据路径类型处理
	if !info.IsDir() {
		// 单文件检查
		base := filepath.Base(searchPath)
		if re.MatchString(base) {
			if fileGlob == "" {
				matched, _ := filepath.Match(fileGlob, base)
				if matched {
					matches = append(matches, SearchMatch{File: searchPath})
				}
			} else {
				matches = append(matches, SearchMatch{File: searchPath})
			}
		}
		return matches, false, nil
	}

	// 遍历目录
	err := filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// 跳过隐藏目录
		if d.IsDir() {
			if shouldSkipDir(path) {
				return fs.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)

		// 检查文件名过滤
		if fileGlob != "" {
			matched, err := filepath.Match(fileGlob, base)
			if err != nil || !matched {
				return nil
			}
		}

		// 检查正则匹配
		if re.MatchString(base) {
			matches = append(matches, SearchMatch{File: path})
			if len(matches) >= maxSearchResults {
				truncated = true
				return fs.SkipAll
			}
		}

		return nil
	})
	if err != nil && err != fs.SkipAll {
		return nil, false, err
	}

	return matches, truncated, nil
}

// shouldSkipFile 判断是否应该跳过该文件（二进制、隐藏文件等）
func shouldSkipFile(path string) bool {
	base := filepath.Base(path)
	// 跳过隐藏文件
	if strings.HasPrefix(base, ".") {
		return true
	}
	// 跳过常见二进制文件扩展名
	binaryExts := map[string]bool{
		".o": true, ".a": true, ".so": true, ".exe": true, ".dll": true,
		".dylib": true, ".zip": true, ".tar": true, ".gz": true,
		".bz2": true, ".xz": true, ".7z": true, ".rar": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".bmp": true, ".ico": true, ".webp": true, ".svg": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mkv": true,
		".mov": true, ".wmv": true, ".flac": true, ".wav": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true,
		".xlsx": true, ".ppt": true, ".pptx": true, ".class": true,
		".jar": true, ".war": true, ".pyc": true, ".pyd": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
		".sqlite": true, ".db": true,
	}
	ext := strings.ToLower(filepath.Ext(base))
	return binaryExts[ext]
}

// shouldSkipDir 判断是否应该跳过该目录
func shouldSkipDir(path string) bool {
	base := filepath.Base(path)
	// 跳过隐藏目录
	if strings.HasPrefix(base, ".") && base != "." {
		return true
	}
	// 跳过常见不需要搜索的目录
	skipDirs := map[string]bool{
		"node_modules": true, "vendor": true, "__pycache__": true,
		".git": true, ".svn": true, ".hg": true,
		"dist": true, "build": true, "out": true, "bin": true,
		".idea": true, ".vscode": true, ".vs": true,
	}
	return skipDirs[base]
}

// isBinaryFile 快速检测文件是否为二进制
func isBinaryFile(file *os.File) bool {
	buf := make([]byte, binaryCheckSize)
	n, _ := file.Read(buf)
	buf = buf[:n]
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}
	return false
}
