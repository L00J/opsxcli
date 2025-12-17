package exec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ForwardCommand 转发命令到系统命令
func ForwardCommand(cmdName string, args []string) error {
	// 查找系统命令
	cmdPath, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("命令 '%s' 未找到，请确保系统已安装该命令", cmdName)
	}

	// 创建命令
	cmd := exec.Command(cmdPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		// 保留原始退出码
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}

// IsCommandAvailable 检查命令是否可用
func IsCommandAvailable(cmdName string) bool {
	_, err := exec.LookPath(cmdName)
	return err == nil
}

// GetCommandPath 获取命令路径
func GetCommandPath(cmdName string) (string, error) {
	return exec.LookPath(cmdName)
}

// ForwardWithFallback 转发命令，如果失败则显示友好提示
func ForwardWithFallback(cmdName string, args []string, fallbackMsg string) error {
	if !IsCommandAvailable(cmdName) {
		if fallbackMsg != "" {
			fmt.Fprintln(os.Stderr, fallbackMsg)
		} else {
			fmt.Fprintf(os.Stderr, "错误: 命令 '%s' 未安装\n", cmdName)
			fmt.Fprintf(os.Stderr, "提示: 请使用系统包管理器安装 '%s'\n", cmdName)
		}
		os.Exit(127) // 命令未找到的标准退出码
	}

	return ForwardCommand(cmdName, args)
}

// BuildCommandHelp 构建命令帮助信息
func BuildCommandHelp(cmdName, short, long string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s - %s\n\n", cmdName, short))

	if long != "" {
		builder.WriteString(long)
		builder.WriteString("\n\n")
	}

	builder.WriteString(fmt.Sprintf("此命令转发到系统的 '%s' 命令。\n", cmdName))
	builder.WriteString(fmt.Sprintf("使用 'man %s' 查看完整文档。\n", cmdName))

	return builder.String()
}
