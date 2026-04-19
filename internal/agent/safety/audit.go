// audit.go - 审计日志持久化
package safety

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// AuditLogWriter 审计日志写入器（JSON Lines 格式）
type AuditLogWriter struct {
	file *os.File
	path string
	mu   sync.Mutex
}

// NewAuditLogWriter 创建审计日志写入器
func NewAuditLogWriter(logPath string) (*AuditLogWriter, error) {
	// 确保目录存在
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("创建审计日志目录失败: %w", err)
	}

	// 以追加模式打开文件，权限 0640
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return nil, fmt.Errorf("打开审计日志文件失败: %w", err)
	}

	return &AuditLogWriter{
		file: f,
		path: logPath,
	}, nil
}

// Write 写入单条审计记录
func (w *AuditLogWriter) Write(record ExecutionRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("序列化审计记录失败: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("写入审计记录失败: %w", err)
	}
	if _, err := w.file.WriteString("\n"); err != nil {
		return fmt.Errorf("写入审计记录换行符失败: %w", err)
	}
	return nil
}

// Close 关闭审计日志文件
func (w *AuditLogWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
