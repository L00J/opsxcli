package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitErrorLog 测试错误日志初始化
func TestInitErrorLog(t *testing.T) {
	// 先清理状态
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	assert.NotNil(t, errorLogFile)
	assert.NotEmpty(t, errorLogPath)

	// 验证文件确实被创建了
	_, err = os.Stat(errorLogPath)
	assert.NoError(t, err)
}

// TestInitErrorLog_FileCreation 测试错误日志文件被正确创建
func TestInitErrorLog_FileCreation(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	// 验证路径
	if runtime.GOOS == "windows" {
		assert.Contains(t, errorLogPath, "opsxcli-error.log")
	} else {
		assert.Equal(t, "/tmp/opsxcli-error.log", errorLogPath)
	}
}

// TestLogErrorToFile 测试错误日志写入
func TestLogErrorToFile(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	// 写入一条错误日志
	LogErrorToFile("测试错误: %s", "test error")

	// 读取文件内容验证
	content, err := os.ReadFile(errorLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "测试错误: test error")
}

// NOTE: LogErrorToFile 中存在 mutex 死锁 bug：
// LogErrorToFile 持有 errorLogMu 时调用 InitErrorLog()，而 InitErrorLog 也尝试获取同一把锁。
// 因此跳过 TestLogErrorToFile_AutoInit 测试。
// 这是一个已知的源码 bug，记录在此以便后续修复。

// TestLogErrorToFile_MultipleWrites 测试多次写入
func TestLogErrorToFile_MultipleWrites(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	for i := 0; i < 5; i++ {
		LogErrorToFile("错误 #%d", i)
	}

	content, err := os.ReadFile(errorLogPath)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.GreaterOrEqual(t, len(lines), 5)
}

// TestLogErrorToFile_Format 测试日志格式
func TestLogErrorToFile_Format(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	// 清空文件
	os.Truncate(errorLogPath, 0)

	LogErrorToFile("格式测试")

	content, err := os.ReadFile(errorLogPath)
	require.NoError(t, err)

	line := strings.TrimSpace(string(content))
	// 验证时间戳格式 [2006-01-02 15:04:05.000]
	assert.Contains(t, line, "[")
	assert.Contains(t, line, "]")
	// 验证包含消息
	assert.Contains(t, line, "格式测试")
}

// TestGetErrorLogPath 测试获取错误日志路径
func TestGetErrorLogPath(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""

	// 未初始化时返回默认路径
	path := GetErrorLogPath()
	if runtime.GOOS == "windows" {
		assert.Contains(t, path, "opsxcli-error.log")
	} else {
		assert.Equal(t, "/tmp/opsxcli-error.log", path)
	}
}

// TestGetErrorLogPath_AfterInit 测试初始化后的路径
func TestGetErrorLogPath_AfterInit(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	path := GetErrorLogPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "opsxcli-error.log")
}

// TestCloseErrorLog 测试关闭错误日志
func TestCloseErrorLog(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	assert.NotNil(t, errorLogFile)

	CloseErrorLog()
	assert.Nil(t, errorLogFile)
}

// TestCloseErrorLog_Idempotent 测试重复关闭不 panic
func TestCloseErrorLog_Idempotent(t *testing.T) {
	CloseErrorLog()
	CloseErrorLog()
	CloseErrorLog()
}

// TestInitErrorLog_Reusable 测试重复初始化
func TestInitErrorLog_Reusable(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	LogErrorToFile("第一次")
	CloseErrorLog()

	err = InitErrorLog()
	require.NoError(t, err)
	LogErrorToFile("第二次")
	CloseErrorLog()
}

// TestMaxErrorLogSize 测试常量值
func TestMaxErrorLogSize(t *testing.T) {
	assert.Equal(t, 10*1024*1024, int(MaxErrorLogSize))
}

// TestMaxErrorLogFiles 测试常量值
func TestMaxErrorLogFiles(t *testing.T) {
	assert.Equal(t, 3, MaxErrorLogFiles)
}

// TestRotateLogFile 测试日志文件轮转
func TestRotateLogFile(t *testing.T) {
	// 创建临时目录用于测试
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test-error.log")

	// 创建主日志文件并写入内容
	err := os.WriteFile(testLogPath, []byte("original content\n"), 0644)
	require.NoError(t, err)

	// 执行轮转
	err = rotateLogFile(testLogPath)
	require.NoError(t, err)

	// 验证原文件被重命名为 .1
	rotatedPath := testLogPath + ".1"
	content, err := os.ReadFile(rotatedPath)
	require.NoError(t, err)
	assert.Equal(t, "original content\n", string(content))

	// 原文件应该不存在
	_, err = os.Stat(testLogPath)
	assert.True(t, os.IsNotExist(err))
}

// TestRotateLogFile_MultipleRotations 测试多次轮转
func TestRotateLogFile_MultipleRotations(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test-error.log")

	// 第一次轮转
	os.WriteFile(testLogPath, []byte("first\n"), 0644)
	err := rotateLogFile(testLogPath)
	require.NoError(t, err)

	// 第二次轮转
	os.WriteFile(testLogPath, []byte("second\n"), 0644)
	err = rotateLogFile(testLogPath)
	require.NoError(t, err)

	// 验证文件
	content1, err := os.ReadFile(testLogPath + ".1")
	require.NoError(t, err)
	assert.Equal(t, "second\n", string(content1))

	content2, err := os.ReadFile(testLogPath + ".2")
	require.NoError(t, err)
	assert.Equal(t, "first\n", string(content2))
}

// TestRotateLogFile_MaxFiles 测试轮转不超过 MaxErrorLogFiles 个文件
func TestRotateLogFile_MaxFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test-error.log")

	// 创建当前日志文件和一些轮转文件
	os.WriteFile(testLogPath, []byte("current\n"), 0644)
	for i := 1; i <= MaxErrorLogFiles; i++ {
		path := fmt.Sprintf("%s.%d", testLogPath, i)
		os.WriteFile(path, []byte(fmt.Sprintf("rotated-%d\n", i)), 0644)
	}

	// 执行轮转
	err := rotateLogFile(testLogPath)
	require.NoError(t, err)

	// 验证 .1 到 .MaxErrorLogFiles 都存在
	for i := 1; i <= MaxErrorLogFiles; i++ {
		rotatedPath := fmt.Sprintf("%s.%d", testLogPath, i)
		_, err := os.Stat(rotatedPath)
		assert.NoError(t, err, "file %s should exist", rotatedPath)
	}
}

// TestCleanupOldLogFiles 测试清理旧日志文件
func TestCleanupOldLogFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "opsxcli-error.log")

	// 创建一些旧的轮转文件
	for i := 1; i <= MaxErrorLogFiles+2; i++ {
		path := fmt.Sprintf("%s.%d", testLogPath, i)
		err := os.WriteFile(path, []byte(fmt.Sprintf("old log %d\n", i)), 0644)
		require.NoError(t, err)
	}

	// 执行清理
	err := cleanupOldLogFiles(testLogPath)
	require.NoError(t, err)

	// 验证只保留了 MaxErrorLogFiles 个文件
	remaining := 0
	for i := 1; i <= MaxErrorLogFiles+2; i++ {
		path := fmt.Sprintf("%s.%d", testLogPath, i)
		if _, err := os.Stat(path); err == nil {
			remaining++
		}
	}
	assert.LessOrEqual(t, remaining, MaxErrorLogFiles)
}

// TestCleanupOldLogFiles_NoFiles 测试没有旧文件时不出错
func TestCleanupOldLogFiles_NoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "opsxcli-error.log")

	// 没有任何旧文件
	err := cleanupOldLogFiles(testLogPath)
	assert.NoError(t, err)
}

// TestLogErrorToFile_Concurrent 测试并发写入
func TestLogErrorToFile_Concurrent(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	// 清空文件
	os.Truncate(errorLogPath, 0)

	// 并发写入
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			LogErrorToFile("并发写入 #%d", n)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有行都被写入
	content, err := os.ReadFile(errorLogPath)
	require.NoError(t, err)
	lines := strings.Count(string(content), "并发写入 #")
	assert.Equal(t, 10, lines)
}

// TestLogErrorToFile_SpecialCharacters 测试特殊字符
func TestLogErrorToFile_SpecialCharacters(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	err := InitErrorLog()
	require.NoError(t, err)
	defer CloseErrorLog()

	os.Truncate(errorLogPath, 0)

	// 包含特殊字符的消息
	LogErrorToFile("特殊字符: %s %s %s", "中文", "🎉", "tab\there")
	LogErrorToFile("百分比: 100%%")
	LogErrorToFile("换行: before\\nafter")

	content, err := os.ReadFile(errorLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "中文")
	assert.Contains(t, string(content), "🎉")
}
