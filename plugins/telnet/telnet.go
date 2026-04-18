package telnet

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
	"opsxcli/internal/logger"
)

// Connect 连接到远程主机（Telnet协议）
func Connect(host, port string, timeout time.Duration, verbose bool) error {
	addr := net.JoinHostPort(host, port)
	if verbose {
		logger.Info("连接到 %s (Telnet)", addr)
	}

	// 建立连接
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		logger.Error("Telnet连接失败 %s: %v", addr, err)
		return fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()

	// 检查是否是真正的终端
	fd := int(os.Stdin.Fd())
	isTerminal := term.IsTerminal(fd)

	// 显示连接成功信息（总是显示，帮助用户知道已连接）
	if isTerminal {
		fmt.Fprintf(os.Stderr, "Connected to %s.\n", addr)
		fmt.Fprintf(os.Stderr, "Escape character is '^C'.\n")
	}

	if verbose {
		logger.Success("已连接到 %s", addr)
	}

	// 只在真正的终端环境下设置原始模式
	var oldState *term.State
	if isTerminal {
		oldState, err = term.MakeRaw(fd)
		if err != nil {
			if verbose {
				logger.Info("无法设置终端原始模式: %v", err)
			}
			isTerminal = false
		} else {
			// 确保退出时恢复终端状态
			defer func() {
				term.Restore(fd, oldState)
				// 输出换行，避免终端显示混乱
				fmt.Println()
			}()
		}
	}

	// 设置信号处理，优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// 创建错误通道
	stdinDone := make(chan error, 1)
	connDone := make(chan error, 1)

	// 从连接读取数据并输出到标准输出
	go func() {
		_, err := io.Copy(os.Stdout, conn)
		connDone <- err
	}()

	// 从标准输入读取数据并发送到连接
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		stdinDone <- err
		// stdin结束后，关闭写入端，通知服务器我们不再发送数据
		// 但继续接收服务器的响应
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
	}()

	// 等待信号或连接关闭
	select {
	case <-sigChan:
		if verbose {
			if !isTerminal {
				fmt.Println()
			}
			logger.Info("收到退出信号，正在断开连接...")
		}
		return nil
	case <-stdinDone:
		// stdin结束了（比如管道输入结束），等待服务器响应完成
		select {
		case <-connDone:
			// 服务器响应也结束了
			return nil
		case <-sigChan:
			return nil
		}
	case err := <-connDone:
		// 服务器关闭连接
		if err != nil && err != io.EOF {
			if verbose {
				logger.Error("连接错误: %v", err)
			}
		}
		return nil
	}
}

// Listen 监听端口（Telnet服务器模式）
func Listen(port int, timeout time.Duration, verbose bool) error {
	if port == 0 {
		logger.Error("Telnet监听模式需要指定端口 (-p)")
		return fmt.Errorf("监听模式需要指定端口 (-p)")
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Error("Telnet监听失败 %s: %v", addr, err)
		return fmt.Errorf("监听失败: %v", err)
	}
	defer listener.Close()

	if verbose {
		logger.Info("Telnet服务器监听 %s", addr)
		fmt.Printf("Listening on :::%d\n", port)
		fmt.Printf("Listening on 0.0.0.0:%d\n", port)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 接受连接
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-sigChan:
					return
				default:
					if verbose {
						logger.Error("接受连接失败: %v", err)
					}
					continue
				}
			}

			if verbose {
				logger.Info("接受来自 %s 的连接", conn.RemoteAddr())
			}

			// 处理每个连接
			go handleConnection(conn, timeout, verbose)
		}
	}()

	// 等待信号
	<-sigChan
	if verbose {
		logger.Info("收到退出信号，正在关闭服务器...")
	}
	return nil
}

// handleConnection 处理Telnet连接
func handleConnection(conn net.Conn, timeout time.Duration, verbose bool) {
	defer conn.Close()

	// 设置超时
	if timeout > 0 {
		conn.SetDeadline(time.Now().Add(timeout))
	}

	// 双向转发
	go func() {
		io.Copy(conn, os.Stdin)
	}()

	io.Copy(os.Stdout, conn)
}
