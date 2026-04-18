package kubernetes

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
	"nhooyr.io/websocket"
)

// ExecPod 在 Pod 中执行命令
func ExecPod(kubeconfigPath, podName, namespace, container string, command []string, stdin bool, tty bool) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// 构建 exec API 路径
	apiPath := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", namespace, podName)
	
	// 构建查询参数
	params := url.Values{}
	for _, cmd := range command {
		params.Add("command", cmd)
	}
	if container != "" {
		params.Add("container", container)
	}
	params.Add("stdin", fmt.Sprintf("%v", stdin))
	params.Add("stdout", "true")
	params.Add("stderr", "true")
	params.Add("tty", fmt.Sprintf("%v", tty))

	// 构建 WebSocket URL (wss://)
	wsURL := client.baseURL + apiPath + "?" + params.Encode()
	wsURL = "wss://" + wsURL[8:] // https:// -> wss://

	// 配置 WebSocket 客户端
	headers := http.Header{}
	if client.token != "" {
		headers.Add("Authorization", "Bearer "+client.token)
	}

	// 创建 WebSocket 连接
	ctx := context.Background()
	tlsConfig := &tls.Config{InsecureSkipVerify: false}
	
	// 从 http.Client 获取 TLS 配置
	if transport, ok := client.httpClient.Transport.(*http.Transport); ok {
		tlsConfig = transport.TLSClientConfig
	}

	wsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: wsClient,
		HTTPHeader: headers,
	})
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 如果是 TTY 模式，设置终端为 raw 模式
	var oldState *term.State
	if tty && stdin {
		oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("设置终端失败: %v", err)
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	// 处理 Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	
	// 启动 goroutine 处理输入输出
	errCh := make(chan error, 2)

	// 从 WebSocket 读取并输出到 stdout/stderr
	go func() {
		for {
			_, message, err := conn.Read(ctx)
			if err != nil {
				// 正常关闭不算错误
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errCh <- nil
					return
				}
				errCh <- err
				return
			}
			
			// Kubernetes exec 协议：第一个字节是流类型
			// 0: stdin, 1: stdout, 2: stderr, 3: error
			if len(message) > 0 {
				streamType := message[0]
				data := message[1:]
				
				switch streamType {
				case 1: // stdout
					os.Stdout.Write(data)
				case 2: // stderr
					os.Stderr.Write(data)
				case 3: // error
					errCh <- fmt.Errorf("exec error: %s", string(data))
					return
				}
			}
		}
	}()

	// 从 stdin 读取并发送到 WebSocket
	if stdin {
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					if err != io.EOF {
						errCh <- err
					}
					return
				}
				
				// 添加流类型前缀 (0 = stdin)
				msg := append([]byte{0}, buf[:n]...)
				err = conn.Write(ctx, websocket.MessageBinary, msg)
				if err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	// 等待完成或信号
	select {
	case <-sigCh:
		return nil
	case err := <-errCh:
		if err == io.EOF {
			return nil
		}
		return err
	}
}
