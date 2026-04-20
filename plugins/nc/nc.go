package nc

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"opsxcli/internal/logger"
)

// 危险shell元字符模式，用于检测命令注入（包括换行符）
var dangerousPatterns = regexp.MustCompile(`[;&|` + "`" + `$(){}\n]`)

// Connect 连接到远程主机
func Connect(host, port string, verbose bool) error {
	addr := net.JoinHostPort(host, port)
	if verbose {
		logger.Info("连接到 %s", addr)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	if verbose {
		logger.Success("已连接到 %s", addr)
	}

	// 双向转发
	go io.Copy(conn, os.Stdin)
	io.Copy(os.Stdout, conn)

	return nil
}

// Listen 监听端口
func Listen(port int, verbose bool, execute string) error {
	if port == 0 {
		err := fmt.Errorf("NC监听模式需要指定端口 (-p)")
		logger.Error("%v", err)
		return err
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	if verbose {
		logger.Info("监听 %s", addr)
		fmt.Printf("Listening on :::%d\n", port)
		fmt.Printf("Listening on 0.0.0.0:%d\n", port)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Error("接受连接失败: %v", err)
			continue
		}

		if verbose {
			logger.Info("接受来自 %s 的连接", conn.RemoteAddr())
		}

		if execute != "" {
			// 执行命令模式
			go handleExecute(conn, execute, verbose)
		} else {
			// 普通转发模式
			go handleConnection(conn, verbose)
		}
	}
}

// handleConnection 处理普通连接
func handleConnection(conn net.Conn, verbose bool) {
	defer conn.Close()

	// 双向转发
	go io.Copy(conn, os.Stdin)
	io.Copy(os.Stdout, conn)
}

// ValidateExecuteCommand 验证执行命令的安全性
// 防止命令注入：拒绝包含shell元字符的命令
func ValidateExecuteCommand(command string) error {
	if command == "" {
		return fmt.Errorf("命令不能为空")
	}
	if dangerousPatterns.MatchString(command) {
		return fmt.Errorf("命令包含危险字符，不允许shell元字符 (;|&|`|$()|{} 等): %q", command)
	}
	return nil
}

// handleExecute 处理执行命令的连接
// 安全改进：使用参数分割代替 /bin/sh -c，避免shell注入
// 注意: 此功能类似nc的-e选项，用于反向shell等场景
func handleExecute(conn net.Conn, command string, verbose bool) {
	defer conn.Close()

	// 安全检查：验证命令不包含危险元字符
	if err := ValidateExecuteCommand(command); err != nil {
		if verbose {
			logger.Error("命令验证失败: %v", err)
		}
		fmt.Fprintf(conn, "Error: %v\n", err)
		return
	}

	// 将命令分割为程序和参数，直接执行而不通过shell
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = conn

	if err := cmd.Run(); err != nil {
		if verbose {
			logger.Error("执行命令失败: %v", err)
		}
	}
}
