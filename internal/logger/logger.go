package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"golang.org/x/term"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	logFile     *os.File

	debugMode  bool
	quietMode  bool
	noColor    bool
)

// Init 初始化日志系统（检测 TTY，自动决定是否启用颜色）
func Init() {
	InitWithOptions(false, false)
}

// InitWithOptions 初始化日志系统
// quiet: 静默模式，抑制 Info/Success/Warning 控制台输出
// noColor: 禁用所有 ANSI 颜色输出（优先于自动检测）
func InitWithOptions(quiet, noColorFlag bool) {
	quietMode = quiet

	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	if noColorFlag || !isTTY {
		noColor = true
		color.NoColor = true
	}

	// 将彩色输出定向到 stderr，避免污染管道数据
	color.Output = os.Stderr

	// 默认不开启调试模式
	debugMode = false

	// 创建日志目录
	logDir := filepath.Join(os.Getenv("HOME"), ".opsxcli", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		initConsoleLoggers()
		return
	}

	// 创建日志文件
	logFileName := filepath.Join(logDir, fmt.Sprintf("opsxcli-%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		initConsoleLoggers()
		return
	}

	logFile = file
	infoLogger = log.New(io.MultiWriter(logFile, os.Stderr), "[INFO] ", log.LstdFlags|log.Lmicroseconds)
	errorLogger = log.New(io.MultiWriter(logFile, os.Stderr), "[ERROR] ", log.LstdFlags|log.Lmicroseconds)
	if debugMode {
		debugLogger = log.New(logFile, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
	}
}

// initConsoleLoggers 初始化控制台logger（无文件时降级）
func initConsoleLoggers() {
	infoLogger = log.New(os.Stderr, "[INFO] ", log.LstdFlags|log.Lmicroseconds)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds)
	if debugMode {
		debugLogger = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
	}
}

// Info 输出信息日志（quiet 模式下抑制）
func Info(format string, v ...interface{}) {
	if quietMode {
		return
	}
	msg := fmt.Sprintf(format, v...)
	if infoLogger != nil {
		infoLogger.Println(msg)
	}
	color.Cyan("ℹ %s", msg)
}

// Error 输出错误日志（quiet 模式下仍输出）
func Error(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	LogErrorToFile(format, v...)
	if errorLogger != nil {
		errorLogger.Println(msg)
	}
	color.Red("✗ %s", msg)
}

// Debug 输出调试日志（quiet 模式下仍输出）
func Debug(format string, v ...interface{}) {
	if !debugMode {
		return
	}
	msg := fmt.Sprintf(format, v...)
	if debugLogger != nil {
		debugLogger.Println(msg)
	}
	color.Yellow("🐛 %s", msg)
}

// Success 输出成功信息（quiet 模式下抑制）
func Success(format string, v ...interface{}) {
	if quietMode {
		return
	}
	msg := fmt.Sprintf(format, v...)
	if infoLogger != nil {
		infoLogger.Printf("[SUCCESS] %s", msg)
	}
	color.Green("✓ %s", msg)
}

// Warning 输出警告信息（quiet 模式下抑制）
func Warning(format string, v ...interface{}) {
	if quietMode {
		return
	}
	msg := fmt.Sprintf(format, v...)
	if infoLogger != nil {
		infoLogger.Printf("[WARN] %s", msg)
	}
	color.Yellow("⚠ %s", msg)
}

// Close 关闭日志文件
func Close() {
	if logFile != nil {
		logFile.Close()
	}
	CloseErrorLog()
}

// SetDebug 设置调试模式
func SetDebug(debug bool) {
	debugMode = debug
	if debug && debugLogger == nil {
		if logFile != nil {
			debugLogger = log.New(logFile, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds)
		}
	}
}

// SetQuiet 设置静默模式
func SetQuiet(quiet bool) {
	quietMode = quiet
}

// SetNoColor 禁用颜色输出
func SetNoColor(noColorFlag bool) {
	noColor = noColorFlag
	color.NoColor = noColorFlag
}

// IsNoColor 返回是否禁用颜色
func IsNoColor() bool {
	return noColor
}

// IsQuiet 返回是否静默模式
func IsQuiet() bool {
	return quietMode
}
