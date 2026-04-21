// transfer.go - 统一传输工具（v0.5.0）
// 合并文件传输接口，本地 cp/远程 scp 自动选择
// LLM 只需调用一个 "transfer" 工具
package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// UnifiedTransferTool 统一传输工具
// 无 host 参数 → 本地文件复制，有 host 参数 → 远程 SCP 传输
type UnifiedTransferTool struct {
	scpTransfer *SCPTransferTool
}

// NewUnifiedTransferTool 创建统一传输工具
func NewUnifiedTransferTool() *UnifiedTransferTool {
	return &UnifiedTransferTool{
		scpTransfer: NewSCPTransferTool(),
	}
}

// Name 返回工具名称
func (t *UnifiedTransferTool) Name() string {
	return "transfer"
}

// Description 返回工具描述（给 LLM 看的）
func (t *UnifiedTransferTool) Description() string {
	return `统一文件传输工具。自动路由到本地或远程传输：
- 不指定 host → 本地文件复制（cp）
- 指定 host + direction=upload → 上传到远程服务器
- 指定 host + direction=download → 从远程服务器下载`
}

// Parameters 返回参数定义（JSON Schema 格式）
func (t *UnifiedTransferTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"source": map[string]interface{}{
				"type":        "string",
				"description": "源文件/目录路径",
			},
			"destination": map[string]interface{}{
				"type":        "string",
				"description": "目标文件/目录路径",
			},
			"direction": map[string]interface{}{
				"type":        "string",
				"description": "传输方向: upload（本地→远程）或 download（远程→本地）。本地复制时忽略此参数",
				"enum":        []string{"upload", "download"},
			},
			"host": map[string]interface{}{
				"type":        "string",
				"description": "远程主机地址（格式: [user@]host[:port]），不指定则本地复制",
			},
		},
		"required": []string{"source", "destination"},
	}
}

// RiskLevel 返回风险等级
func (t *UnifiedTransferTool) RiskLevel() RiskLevel {
	return RiskMedium
}

// Execute 执行文件传输，自动路由本地/远程
func (t *UnifiedTransferTool) Execute(ctx context.Context, args map[string]interface{}) (*Result, error) {
	source := parseStringParam(args, "source")
	if source == "" {
		return &Result{
			Success: false,
			Error:   "source 参数不能为空",
		}, fmt.Errorf("source 参数不能为空")
	}

	destination := parseStringParam(args, "destination")
	if destination == "" {
		return &Result{
			Success: false,
			Error:   "destination 参数不能为空",
		}, fmt.Errorf("destination 参数不能为空")
	}

	host := parseStringParam(args, "host")

	if host == "" {
		// 本地文件复制
		return t.localCopy(ctx, source, destination)
	}

	// 远程传输 — 委托给 SCPTransferTool
	// 确保 host 和 direction 参数在 args 中
	remoteArgs := make(map[string]interface{})
	for k, v := range args {
		remoteArgs[k] = v
	}
	remoteArgs["host"] = host

	direction := parseStringParam(args, "direction")
	if direction == "" {
		return &Result{
			Success: false,
			Error:   "远程传输需要指定 direction 参数 (upload 或 download)",
		}, fmt.Errorf("远程传输需要指定 direction 参数")
	}

	return t.scpTransfer.Execute(ctx, remoteArgs)
}

// localCopy 本地文件复制
func (t *UnifiedTransferTool) localCopy(ctx context.Context, source, destination string) (*Result, error) {
	start := time.Now()

	// 检查上下文
	if ctx.Err() != nil {
		return &Result{
			Success: false,
			Error:   "操作已取消",
		}, ctx.Err()
	}

	// 获取源文件信息
	srcInfo, err := os.Stat(source)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("无法访问源路径: %v", err),
		}, err
	}

	// 处理目录复制
	if srcInfo.IsDir() {
		return t.copyDirectory(ctx, source, destination, start)
	}

	// 确保目标目录存在
	dstDir := filepath.Dir(destination)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建目标目录失败: %v", err),
		}, err
	}

	// 打开源文件
	srcFile, err := os.Open(source)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("打开源文件失败: %v", err),
		}, err
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(destination)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("创建目标文件失败: %v", err),
		}, err
	}
	defer dstFile.Close()

	// 复制内容
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("复制文件失败: %v", err),
		}, err
	}

	// 保留源文件权限
	if err := os.Chmod(destination, srcInfo.Mode()); err != nil {
		// 权限设置失败不影响主流程
		_ = err
	}

	duration := time.Since(start)
	return &Result{
		Success: true,
		Output:  fmt.Sprintf("已复制: %s → %s (%s, 耗时 %v)", source, destination, formatFileSize(written), duration.Round(time.Millisecond)),
		Summary: fmt.Sprintf("[本地] 文件复制成功 (%s)", formatFileSize(written)),
	}, nil
}

// copyDirectory 递归复制目录
func (t *UnifiedTransferTool) copyDirectory(ctx context.Context, source, destination string, start time.Time) (*Result, error) {
	var totalSize int64
	var fileCount int

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(destination, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 确保目标目录存在
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		// 复制文件
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		written, err := io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}

		totalSize += written
		fileCount++
		return os.Chmod(dstPath, info.Mode())
	})

	if err != nil {
		return &Result{
			Success: false,
			Error:   fmt.Sprintf("复制目录失败: %v", err),
		}, err
	}

	duration := time.Since(start)
	return &Result{
		Success: true,
		Output:  fmt.Sprintf("已复制目录: %s → %s (%d 个文件, %s, 耗时 %v)", source, destination, fileCount, formatFileSize(totalSize), duration.Round(time.Millisecond)),
		Summary: fmt.Sprintf("[本地] 目录复制成功 (%d 文件, %s)", fileCount, formatFileSize(totalSize)),
	}, nil
}
