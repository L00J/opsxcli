package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

const (
	// MaxErrorLogSize 错误日志文件最大大小（10MB）
	MaxErrorLogSize = 10 * 1024 * 1024
	// MaxErrorLogFiles 最多保留的历史文件数量
	MaxErrorLogFiles = 3
)

var (
	errorLogFile *os.File
	errorLogMu   sync.Mutex
	errorLogPath string
)

// InitErrorLog 初始化错误日志文件
// 注意：此函数内部获取 errorLogMu 锁，不可在持锁状态下调用。
func InitErrorLog() error {
	errorLogMu.Lock()
	defer errorLogMu.Unlock()

	return initErrorLogLocked()
}

// initErrorLogLocked 在已持锁状态下初始化错误日志文件
func initErrorLogLocked() error {
	// 获取临时目录（跨平台）
	tmpDir := os.TempDir()

	// 确定错误日志路径
	if runtime.GOOS == "windows" {
		// Windows: 使用 %TEMP%\opsxcli-error.log
		errorLogPath = filepath.Join(tmpDir, "opsxcli-error.log")
	} else {
		// Linux/macOS: 使用 /tmp/opsxcli-error.log
		errorLogPath = "/tmp/opsxcli-error.log"
	}

	// 打开或创建错误日志文件
	file, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("无法创建错误日志文件 %s: %v", errorLogPath, err)
	}

	errorLogFile = file
	return nil
}

// LogErrorToFile 记录错误到统一错误日志文件
func LogErrorToFile(format string, v ...interface{}) {
	errorLogMu.Lock()
	defer errorLogMu.Unlock()

	if errorLogFile == nil {
		// 如果未初始化，尝试初始化（使用 locked 版本避免死锁）
		if err := initErrorLogLocked(); err != nil {
			// 如果初始化失败，只输出到stderr
			fmt.Fprintf(os.Stderr, "[ERROR] %s\n", fmt.Sprintf(format, v...))
			return
		}
	}

	// 获取调用者信息（跳过 LogErrorToFile 和 Error 函数本身）
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		// 如果获取失败，尝试获取直接调用者
		_, file, line, ok = runtime.Caller(2)
		if !ok {
			file = "unknown"
			line = 0
		}
	}

	// 只保留文件名
	fileName := filepath.Base(file)

	// 格式化错误日志
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, v...)
	logLine := fmt.Sprintf("[%s] [%s:%d] %s\n", timestamp, fileName, line, message)

	// 写入错误日志文件
	if _, err := errorLogFile.WriteString(logLine); err != nil {
		// 如果写入失败，输出到stderr
		fmt.Fprintf(os.Stderr, "[ERROR] 写入错误日志失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", message)
		return
	}

	// 立即刷新到磁盘，确保错误日志被写入
	if err := errorLogFile.Sync(); err != nil {
		// Sync 失败不影响主流程，只记录到 stderr
		fmt.Fprintf(os.Stderr, "[WARN] 刷新错误日志失败: %v\n", err)
	}

	// 检查文件大小，如果超过限制则轮转（在锁内执行）
	fileSize := int64(0)
	if info, err := errorLogFile.Stat(); err == nil {
		fileSize = info.Size()
	}

	// 如果文件大小超过限制，需要轮转（在锁内安全执行）
	if fileSize >= MaxErrorLogSize {
		if err := checkAndRotateLogLocked(); err != nil {
			// 轮转失败不影响主流程，只记录到 stderr
			fmt.Fprintf(os.Stderr, "[WARN] 日志轮转失败: %v\n", err)
		}
	}
}

// GetErrorLogPath 获取错误日志文件路径
func GetErrorLogPath() string {
	if errorLogPath != "" {
		return errorLogPath
	}

	// 如果未初始化，返回默认路径
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "opsxcli-error.log")
	}
	return "/tmp/opsxcli-error.log"
}

// CloseErrorLog 关闭错误日志文件
func CloseErrorLog() {
	errorLogMu.Lock()
	defer errorLogMu.Unlock()

	if errorLogFile != nil {
		errorLogFile.Close()
		errorLogFile = nil
	}
}

// checkAndRotateLogLocked 检查日志文件大小，如果超过限制则轮转（必须在持锁状态下调用）
func checkAndRotateLogLocked() error {
	if errorLogFile == nil || errorLogPath == "" {
		return nil
	}

	// 获取当前文件大小
	info, err := errorLogFile.Stat()
	if err != nil {
		return fmt.Errorf("获取日志文件信息失败: %v", err)
	}

	// 如果文件大小未超过限制，不需要轮转
	if info.Size() < MaxErrorLogSize {
		return nil
	}

	// 关闭当前文件
	if err := errorLogFile.Close(); err != nil {
		return fmt.Errorf("关闭日志文件失败: %v", err)
	}
	errorLogFile = nil

	// 轮转日志文件
	if err := rotateLogFile(errorLogPath); err != nil {
		return fmt.Errorf("轮转日志文件失败: %v", err)
	}

	// 重新打开日志文件
	file, err := os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("重新打开日志文件失败: %v", err)
	}

	errorLogFile = file
	return nil
}

// rotateLogFile 轮转日志文件
func rotateLogFile(logPath string) error {
	// 轮转现有文件：opsxcli-error.log -> opsxcli-error.log.1
	// opsxcli-error.log.1 -> opsxcli-error.log.2
	// opsxcli-error.log.2 -> opsxcli-error.log.3
	// 删除 opsxcli-error.log.3（如果存在）

	// 先删除最旧的文件（如果存在）
	oldestFile := fmt.Sprintf("%s.%d", logPath, MaxErrorLogFiles)
	if _, err := os.Stat(oldestFile); err == nil {
		if err := os.Remove(oldestFile); err != nil {
			return fmt.Errorf("删除最旧日志文件失败: %v", err)
		}
	}

	// 从后往前轮转文件
	for i := MaxErrorLogFiles - 1; i >= 1; i-- {
		oldFile := fmt.Sprintf("%s.%d", logPath, i)
		newFile := fmt.Sprintf("%s.%d", logPath, i+1)

		// 如果旧文件存在，重命名为新文件
		if _, err := os.Stat(oldFile); err == nil {
			if err := os.Rename(oldFile, newFile); err != nil {
				return fmt.Errorf("重命名日志文件失败 %s -> %s: %v", oldFile, newFile, err)
			}
		}
	}

	// 将当前日志文件重命名为 .1
	rotatedFile := fmt.Sprintf("%s.1", logPath)
	if _, err := os.Stat(logPath); err == nil {
		if err := os.Rename(logPath, rotatedFile); err != nil {
			return fmt.Errorf("重命名当前日志文件失败: %v", err)
		}
	}

	// 清理超过保留数量的旧日志文件
	if err := cleanupOldLogFiles(logPath); err != nil {
		// 清理失败不影响轮转，只记录警告
		fmt.Fprintf(os.Stderr, "[WARN] 清理旧日志文件失败: %v\n", err)
	}

	return nil
}

// cleanupOldLogFiles 清理超过保留数量的旧日志文件（在轮转时调用）
func cleanupOldLogFiles(logPath string) error {
	// 获取日志目录
	logDir := filepath.Dir(logPath)
	baseName := filepath.Base(logPath)

	// 读取目录中的所有文件
	files, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("读取日志目录失败: %v", err)
	}

	// 查找所有相关的日志文件（.1, .2, .3 等）
	var logFiles []string
	for _, file := range files {
		name := file.Name()
		// 匹配 opsxcli-error.log.N 格式
		if len(name) > len(baseName) && name[:len(baseName)] == baseName && name[len(baseName)] == '.' {
			// 检查是否是数字后缀
			suffix := name[len(baseName)+1:]
			isNumber := true
			for _, c := range suffix {
				if c < '0' || c > '9' {
					isNumber = false
					break
				}
			}
			if isNumber {
				fullPath := filepath.Join(logDir, name)
				logFiles = append(logFiles, fullPath)
			}
		}
	}

	// 如果文件数量不超过限制，不需要清理
	if len(logFiles) <= MaxErrorLogFiles {
		return nil
	}

	// 按文件名排序（.1, .2, .3 等）
	sort.Strings(logFiles)

	// 删除超过限制的文件（保留最新的 MaxErrorLogFiles 个）
	filesToDelete := logFiles[:len(logFiles)-MaxErrorLogFiles]
	for _, file := range filesToDelete {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("删除旧日志文件失败 %s: %v", file, err)
		}
	}

	return nil
}
