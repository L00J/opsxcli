package nc

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestValidateExecuteCommand_SafeCommands 验证安全的命令通过验证
func TestValidateExecuteCommand_SafeCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"simple binary", "/bin/sh", false},
		{"with simple args", "/bin/cat /etc/hosts", false},
		{"echo hello", "echo hello", false},
		{"path with dashes", "/usr/bin/python3 -c test", false},
		{"empty command", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExecuteCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecuteCommand(%q) error = %v, wantErr %v", tt.command, err, tt.wantErr)
			}
		})
	}
}

// TestValidateExecuteCommand_CommandInjection 验证命令注入攻击被阻止
// 回归测试：确保C1安全漏洞（命令注入）已被修复
func TestValidateExecuteCommand_CommandInjection(t *testing.T) {
	injectionTests := []struct {
		name    string
		command string
	}{
		{"semicolon chaining", "/bin/sh; rm -rf /"},
		{"double ampersand", "/bin/sh && cat /etc/passwd"},
		{"pipe injection", "/bin/sh | tee /tmp/stolen"},
		{"backtick injection", "echo `cat /etc/passwd`"},
		{"dollar substitution", "echo $(cat /etc/passwd)"},
		{"curly brace", "echo {cat,/etc/passwd}"},
		{"OR operator", "/bin/sh || /bin/bash"},
		{"newline injection", "/bin/sh\nrm -rf /"},
		{"mixed injection", "/bin/sh; cat /etc/passwd | tee /tmp/out"},
	}

	for _, tt := range injectionTests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExecuteCommand(tt.command)
			if err == nil {
				t.Errorf("ValidateExecuteCommand(%q) should reject command injection, but it passed", tt.command)
			}
		})
	}
}

// TestConnect_Success 测试成功连接
func TestConnect_Success(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		conn.Write([]byte("hello\n"))
		conn.Close()
	}()

	err = Connect("127.0.0.1", fmt.Sprintf("%d", addr.Port), false)
	if err != nil {
		t.Errorf("Connect() error = %v", err)
	}
}

// TestConnect_Refused 测试连接被拒绝
func TestConnect_Refused(t *testing.T) {
	err := Connect("127.0.0.1", "1", false)
	if err == nil {
		t.Error("Connect() should fail on refused connection")
	}
}

// TestHandleExecute_RejectsInjection 集成测试：handleExecute拒绝注入命令
func TestHandleExecute_RejectsInjection(t *testing.T) {
	// 启动本地TCP服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	addr := listener.Addr().(*net.TCPAddr)

	// 在goroutine中启动Accept循环，只接受一个连接
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		// handleExecute 应拒绝包含注入的命令 (verbose=false避免logger死锁)
		handleExecute(conn, "/bin/sh; rm -rf /", false)
		listener.Close()
	}()

	// 给监听器启动时间
	time.Sleep(50 * time.Millisecond)

	// 连接到监听器
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", addr.Port), 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// 读取响应 - 应包含错误信息
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := conn.Read(buf)
	response := string(buf[:n])

	if !strings.Contains(response, "Error") {
		t.Errorf("Expected rejection error message, got: %q", response)
	}
}

// TestHandleExecute_SafeCommand 测试安全命令可以执行
func TestHandleExecute_SafeCommand(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	addr := listener.Addr().(*net.TCPAddr)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		// 使用安全的 echo 命令
		handleExecute(conn, "echo test_output_123", true)
		listener.Close()
	}()

	time.Sleep(50 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", addr.Port), 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := conn.Read(buf)
	response := string(buf[:n])

	if !strings.Contains(response, "test_output_123") {
		t.Errorf("Expected 'test_output_123' in output, got: %q", response)
	}
}
