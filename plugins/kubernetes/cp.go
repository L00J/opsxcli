package kubernetes

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"nhooyr.io/websocket"
)

// CopyToPod 复制文件到 Pod
func CopyToPod(kubeconfigPath, podName, namespace, container, srcPath, destPath string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// 读取源文件
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer srcFile.Close()

	// 处理目标路径:如果以 / 结尾,说明是目录,需要添加文件名
	if strings.HasSuffix(destPath, "/") {
		destPath = destPath + filepath.Base(srcPath)
	}

	// 使用 dd 命令写入文件
	// 虽然依赖 Pod 中的 dd,但几乎所有 Linux 容器都有 dd
	// dd 比 cat+shell 重定向更可靠,处理二进制数据更好
	command := []string{"dd", "of=" + destPath}

	return execWithStdin(client, podName, namespace, container, command, srcFile)
}

// CopyFromPod 从 Pod 复制文件到本地
func CopyFromPod(kubeconfigPath, podName, namespace, container, srcPath, destPath string) error {
	client, err := NewK8sHTTPClient(kubeconfigPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = "default"
	}

	// 创建本地文件
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer destFile.Close()

	// 使用 cat 命令读取文件内容
	command := []string{"cat", srcPath}

	return execWithStdout(client, podName, namespace, container, command, destFile)
}

// execWithStdin 执行命令并传递 stdin
func execWithStdin(client *K8sHTTPClient, podName, namespace, container string, command []string, stdin io.Reader) error {
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
	params.Add("stdin", "true")
	params.Add("stdout", "true")
	params.Add("stderr", "true")
	params.Add("tty", "false")

	// 构建 WebSocket URL
	wsURL := client.baseURL + apiPath + "?" + params.Encode()
	wsURL = "wss://" + wsURL[8:]

	// 配置 WebSocket 客户端
	headers := http.Header{}
	if client.token != "" {
		headers.Add("Authorization", "Bearer "+client.token)
	}

	ctx := context.Background()
	tlsConfig := &tls.Config{InsecureSkipVerify: false}

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

	errCh := make(chan error, 2)

	// 从 WebSocket 读取并输出错误
	go func() {
		for {
			_, message, err := conn.Read(ctx)
			if err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
					errCh <- nil
					return
				}
				errCh <- err
				return
			}

			if len(message) > 0 {
				streamType := message[0]
				data := message[1:]

				switch streamType {
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
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := stdin.Read(buf)
			if n > 0 {
				// 添加流类型前缀 (0 = stdin)
				msg := append([]byte{0}, buf[:n]...)
				if err := conn.Write(ctx, websocket.MessageBinary, msg); err != nil {
					errCh <- err
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					// stdin 读取完成,不发送错误,等待命令执行完成
					return
				} else {
					errCh <- err
					return
				}
			}
		}
	}()

	// 只等待读取 goroutine 完成(命令执行完毕)
	return <-errCh
}

// execWithStdout 执行命令并捕获 stdout
func execWithStdout(client *K8sHTTPClient, podName, namespace, container string, command []string, stdout io.Writer) error {
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
	params.Add("stdin", "false")
	params.Add("stdout", "true")
	params.Add("stderr", "true")
	params.Add("tty", "false")

	// 构建 WebSocket URL
	wsURL := client.baseURL + apiPath + "?" + params.Encode()
	wsURL = "wss://" + wsURL[8:]

	// 配置 WebSocket 客户端
	headers := http.Header{}
	if client.token != "" {
		headers.Add("Authorization", "Bearer "+client.token)
	}

	ctx := context.Background()
	tlsConfig := &tls.Config{InsecureSkipVerify: false}

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

	// 从 WebSocket 读取
	for {
		_, message, err := conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return nil
			}
			return err
		}

		if len(message) > 0 {
			streamType := message[0]
			data := message[1:]

			switch streamType {
			case 1: // stdout
				if _, err := stdout.Write(data); err != nil {
					return err
				}
			case 2: // stderr
				os.Stderr.Write(data)
			case 3: // error
				return fmt.Errorf("exec error: %s", string(data))
			}
		}
	}
}

// CopyFile 智能复制文件（自动判断方向）
func CopyFile(kubeconfigPath, srcPath, destPath, namespace, container string) error {
	// 判断是从 Pod 复制到本地，还是从本地复制到 Pod
	// 格式: pod:/path 或 /local/path

	srcPod, srcFile := parsePodPath(srcPath)
	destPod, destFile := parsePodPath(destPath)

	if srcPod != "" && destPod != "" {
		return fmt.Errorf("不支持 Pod 之间直接复制")
	}

	if srcPod == "" && destPod == "" {
		return fmt.Errorf("至少需要一个 Pod 路径 (格式: pod-name:/path)")
	}

	if srcPod != "" {
		// 从 Pod 复制到本地
		return CopyFromPod(kubeconfigPath, srcPod, namespace, container, srcFile, destFile)
	}

	// 从本地复制到 Pod
	return CopyToPod(kubeconfigPath, destPod, namespace, container, srcFile, destFile)
}

// parsePodPath 解析 Pod 路径格式 "pod-name:/path"
func parsePodPath(path string) (pod string, file string) {
	parts := strings.SplitN(path, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", path
}
