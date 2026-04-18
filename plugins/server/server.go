package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"opsxcli/internal/logger"
)

// Start 启动HTTP文件服务器
// password 格式：username:password，如果为空则不启用认证
func Start(host string, port int, directory string, password string) error {
	if directory == "" {
		// 默认使用当前目录
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("获取当前目录失败: %v", err)
		}
	}

	// 检查目录是否存在
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("解析目录路径失败: %v", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("目录不存在: %s", absDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("路径不是目录: %s", absDir)
	}

	// 创建文件服务器
	fileServer := http.FileServer(http.Dir(absDir))

	// 解析认证信息
	var authUsername, authPassword string
	if password != "" {
		parts := strings.SplitN(password, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("密码格式错误，应为 username:password")
		}
		authUsername = parts[0]
		authPassword = parts[1]
		if authUsername == "" || authPassword == "" {
			return fmt.Errorf("用户名和密码不能为空")
		}
		logger.Info("已启用HTTP基本认证，用户名: %s", authUsername)
	}

	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 如果启用了认证，检查认证信息
		if password != "" {
			if !checkBasicAuth(r, authUsername, authPassword) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				w.WriteHeader(http.StatusUnauthorized)
				logger.Error("认证失败: %s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
				fmt.Fprintf(w, "401 Unauthorized\n")
				return
			}
		}

		// 记录访问日志
		logger.Info("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
		fileServer.ServeHTTP(w, r)
	})

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 设置信号处理，优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 在goroutine中启动服务器
	go func() {
		logger.Success("HTTP服务器启动在 %s", addr)
		logger.Info("服务目录: %s", absDir)

		// 显示访问地址
		if host == "0.0.0.0" || host == "" {
			// 获取本机IP地址
			localIP := getLocalIP()
			logger.Info("本地访问: http://localhost:%d", port)
			if localIP != "" {
				logger.Info("外部访问: http://%s:%d", localIP, port)
			} else {
				logger.Info("外部访问: http://%s:%d (或使用本机IP)", host, port)
			}
		} else if host == "127.0.0.1" || host == "localhost" {
			logger.Info("访问地址: http://%s:%d (仅本地访问)", host, port)
		} else {
			logger.Info("访问地址: http://%s:%d", host, port)
		}
		logger.Info("按 Ctrl+C 停止服务器")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-sigChan
	logger.Info("收到退出信号，正在关闭服务器...")

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("服务器关闭失败: %v", err)
		return err
	}

	logger.Success("服务器已关闭")
	return nil
}

// checkBasicAuth 检查HTTP基本认证
func checkBasicAuth(r *http.Request, username, password string) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}

	// 解析 Basic 认证
	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	// 解码 base64
	encoded := strings.TrimPrefix(auth, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}

	// 解析 username:password
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}

	return parts[0] == username && parts[1] == password
}

// getLocalIP 获取本机IP地址
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
