package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckAndRotateLogLocked_NilFile 测试 nil 文件时返回 nil
func TestCheckAndRotateLogLocked_NilFile(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	errorLogFile = nil

	// errorLogFile 为 nil，应该返回 nil
	err := checkAndRotateLogLocked()
	assert.NoError(t, err)
}

// TestCheckAndRotateLogLocked_EmptyPath 测试空路径时返回 nil
func TestCheckAndRotateLogLocked_EmptyPath(t *testing.T) {
	CloseErrorLog()
	errorLogPath = ""
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-log-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	errorLogFile = tmpFile
	err = checkAndRotateLogLocked()
	assert.NoError(t, err)
}

// TestCheckAndRotateLogLocked_SmallFile 测试小文件不需要轮转
func TestCheckAndRotateLogLocked_SmallFile(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test-rotate.log")

	file, err := os.OpenFile(testLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)
	defer file.Close()

	// 写入小量数据
	file.WriteString("small content\n")

	CloseErrorLog()
	errorLogFile = file
	errorLogPath = testLogPath

	err = checkAndRotateLogLocked()
	assert.NoError(t, err)

	// 文件应该还是同一个（没有被轮转）
	_, err = os.Stat(testLogPath)
	assert.NoError(t, err)
}

// TestCheckAndRotateLogLocked_LargeFile 测试大文件需要轮转
func TestCheckAndRotateLogLocked_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test-rotate-large.log")

	file, err := os.OpenFile(testLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)

	// 写入超过 MaxErrorLogSize 的数据
	largeData := strings.Repeat("x", MaxErrorLogSize+1024)
	file.WriteString(largeData)
	file.Close()

	// 重新以 append 模式打开
	file, err = os.OpenFile(testLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)

	CloseErrorLog()
	errorLogFile = file
	errorLogPath = testLogPath

	err = checkAndRotateLogLocked()
	assert.NoError(t, err)

	// 验证 .1 文件被创建（包含旧数据）
	rotatedPath := testLogPath + ".1"
	_, err = os.Stat(rotatedPath)
	assert.NoError(t, err, "rotated file .1 should exist")

	// 新文件应该存在且更小
	info, err := os.Stat(testLogPath)
	if err == nil {
		assert.Less(t, info.Size(), int64(MaxErrorLogSize))
	}
}

// TestInitConsoleLoggers 测试控制台 logger 初始化
func TestInitConsoleLoggers(t *testing.T) {
	Init()
	defer Close()

	// 直接调用 initConsoleLoggers（已在 Init 内部被覆盖调用）
	initConsoleLoggers()

	assert.NotNil(t, infoLogger)
	assert.NotNil(t, errorLogger)
}

// TestInfo_QuietMode 测试 quiet 模式下 Info 不输出
func TestInfo_QuietMode(t *testing.T) {
	InitWithOptions(true, true)
	defer Close()

	// 应该不 panic
	Info("这条消息不应该输出: %s", "test")
}

// TestInfo_NormalMode 测试 Info 正常输出
func TestInfo_NormalMode(t *testing.T) {
	InitWithOptions(false, true)
	defer Close()

	Info("普通信息消息: %s", "test")
}

// TestError_Output 测试 Error 输出
func TestError_Output(t *testing.T) {
	InitWithOptions(false, true)
	defer Close()

	Error("错误消息: %s", "test error")
}

// TestDebug_Enabled 测试 Debug 启用
func TestDebug_Enabled(t *testing.T) {
	InitWithOptions(false, true)
	SetDebug(true)
	defer Close()

	Debug("调试消息: %s", "test debug")
}

// TestDebug_Disabled 测试 Debug 禁用
func TestDebug_Disabled(t *testing.T) {
	InitWithOptions(false, true)
	SetDebug(false)
	defer Close()

	Debug("这条消息不应该输出: %s", "test")
}

// TestSuccess_QuietMode 测试 quiet 模式下 Success 不输出
func TestSuccess_QuietMode(t *testing.T) {
	InitWithOptions(true, true)
	defer Close()

	Success("这条消息不应该输出: %s", "test")
}

// TestSuccess_NormalMode 测试 Success 正常输出
func TestSuccess_NormalMode(t *testing.T) {
	InitWithOptions(false, true)
	defer Close()

	Success("成功消息: %s", "test")
}

// TestWarning_QuietMode 测试 quiet 模式下 Warning 不输出
func TestWarning_QuietMode(t *testing.T) {
	InitWithOptions(true, true)
	defer Close()

	Warning("这条消息不应该输出: %s", "test")
}

// TestWarning_NormalMode 测试 Warning 正常输出
func TestWarning_NormalMode(t *testing.T) {
	InitWithOptions(false, true)
	defer Close()

	Warning("警告消息: %s", "test")
}

// TestSetDebug_WithLogFile 测试已有日志文件时设置 Debug
func TestSetDebug_WithLogFile(t *testing.T) {
	Init()
	defer Close()

	SetDebug(true)
	assert.True(t, debugMode)
	assert.NotNil(t, debugLogger)

	SetDebug(false)
	assert.False(t, debugMode)
}

// TestInitWithOptions_NoColor 测试无颜色初始化
func TestInitWithOptions_NoColor(t *testing.T) {
	InitWithOptions(false, true)
	defer Close()

	assert.True(t, noColor)
}

// TestLogErrorToFile_TriggerRotation 测试通过写入触发日志轮转
func TestLogErrorToFile_TriggerRotation(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "test-rotation.log")

	file, err := os.OpenFile(testLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)

	// 写入大量数据到接近大小限制
	largeData := strings.Repeat("x", MaxErrorLogSize-100)
	file.WriteString(largeData)

	CloseErrorLog()
	errorLogFile = file
	errorLogPath = testLogPath

	// 这条日志应该触发轮转检查（写入后文件大小 > MaxErrorLogSize）
	LogErrorToFile("触发轮转测试")

	// 验证文件仍然可用
	_, err = os.Stat(testLogPath)
	assert.NoError(t, err)
}

// TestCleanupOldLogFiles_WithNonNumericSuffix 测试清理不匹配数字后缀的文件
func TestCleanupOldLogFiles_WithNonNumericSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	testLogPath := filepath.Join(tmpDir, "opsxcli-error.log")

	// 创建一个非数字后缀的文件
	nonNumericPath := testLogPath + ".bak"
	err := os.WriteFile(nonNumericPath, []byte("backup\n"), 0644)
	require.NoError(t, err)

	// 创建正常的数字后缀文件
	for i := 1; i <= 2; i++ {
		path := testLogPath + "." + string(rune('0'+i))
		err := os.WriteFile(path, []byte("log\n"), 0644)
		require.NoError(t, err)
	}

	err = cleanupOldLogFiles(testLogPath)
	assert.NoError(t, err)

	// 非数字文件应该保留
	_, err = os.Stat(nonNumericPath)
	assert.NoError(t, err)
}
