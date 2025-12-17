package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	logFile     *os.File
	debugMode   bool
)

// Init 初始化日志系统
func Init() {
	// 默认不开启调试模式
	InitWithDebug(false)

	// 初始化统一错误日志（忽略错误，如果失败会在 LogErrorToFile 中重试）
	_ = InitErrorLog()
}

// InitWithDebug 初始化日志系统（带调试模式）
func InitWithDebug(debug bool) {
	debugMode = debug

	// 创建日志目录
	logDir := filepath.Join(os.Getenv("HOME"), ".opsxcli", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// 如果无法创建日志目录，只使用控制台输出
		initConsoleLoggers()
		return
	}

	// 创建日志文件
	logFileName := filepath.Join(logDir, fmt.Sprintf("opsxcli-%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// 如果无法创建日志文件，只使用控制台输出
		initConsoleLoggers()
		return
	}

	logFile = file

	// 创建多个logger实例
	infoLogger = log.New(logFile, "[INFO] ", log.LstdFlags|log.Lmicroseconds)
	errorLogger = log.New(logFile, "[ERROR] ", log.LstdFlags|log.Lmicroseconds)
	if debugMode {
		debugLogger = log.New(logFile, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
	}
}

// initConsoleLoggers 初始化控制台logger
func initConsoleLoggers() {
	infoLogger = log.New(os.Stdout, "", 0)
	errorLogger = log.New(os.Stderr, "", 0)
	if debugMode {
		debugLogger = log.New(os.Stdout, "", 0)
	}
}

// Info 输出信息日志
func Info(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if infoLogger != nil {
		infoLogger.Println(msg)
	}
	color.Cyan("ℹ %s\n", msg)
}

// Error 输出错误日志（同时记录到统一错误日志文件）
func Error(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)

	// 记录到统一错误日志文件
	LogErrorToFile(format, v...)

	// 记录到常规日志文件
	if errorLogger != nil {
		errorLogger.Println(msg)
	}

	// 输出到控制台
	color.Red("✗ %s\n", msg)
}

// Debug 输出调试日志
func Debug(format string, v ...interface{}) {
	if !debugMode {
		return
	}
	msg := fmt.Sprintf(format, v...)
	if debugLogger != nil {
		debugLogger.Println(msg)
	}
	color.Yellow("🐛 %s\n", msg)
}

// Success 输出成功信息
func Success(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if infoLogger != nil {
		infoLogger.Printf("[SUCCESS] %s\n", msg)
	}
	color.Green("✓ %s\n", msg)
}

// Close 关闭日志文件
func Close() {
	if logFile != nil {
		logFile.Close()
	}
	// 关闭错误日志文件
	CloseErrorLog()
}

// SetDebug 设置调试模式
func SetDebug(debug bool) {
	debugMode = debug
	if debug && debugLogger == nil {
		if logFile != nil {
			debugLogger = log.New(logFile, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
		} else {
			debugLogger = log.New(os.Stdout, "", 0)
		}
	}
}
