package request

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"opsxcli/internal/logger"
	"opsxcli/internal/tui"
)

// Send 发送HTTP请求
func Send(urlStr, method string, headers []string, data, output string) error {
	return SendWithStatus(urlStr, method, headers, data, output, true)
}

// SendWithStatus 发送HTTP请求（可选是否显示状态）
func SendWithStatus(urlStr, method string, headers []string, data, output string, showStatus bool) error {
	// 解析URL获取域名
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	// 创建工具状态提示
	var status *tui.ToolStatus

	if showStatus {
		status = tui.NewToolStatus()
		// 显示请求信息
		statusMsg := fmt.Sprintf("%s %s", method, parsedURL.Host)
		status.StartWithMessage("request", statusMsg)
	}

	// 创建请求
	var body io.Reader
	if data != "" {
		body = bytes.NewBufferString(data)
	}

	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		if showStatus {
			status.Stop(false, err.Error())
		}
		return err
	}

	// 设置请求头
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	// 如果没有Content-Type，根据数据自动设置
	if data != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		if showStatus {
			status.Stop(false, err.Error())
		}
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if showStatus {
			status.Stop(false, err.Error())
		}
		return err
	}

	// 停止状态提示
	if showStatus {
		statusCode := resp.StatusCode
		success := statusCode >= 200 && statusCode < 400

		// 构建状态消息
		var statusMsg string
		if output != "" {
			statusMsg = fmt.Sprintf("%d - 已保存到 %s", statusCode, output)
		} else {
			// 计算响应大小
			size := len(respBody)
			sizeStr := formatBytes(size)
			statusMsg = fmt.Sprintf("%d - %s", statusCode, sizeStr)
		}

		status.Stop(success, statusMsg)

		// 额外显示详细信息（可选）
		if !success {
			logger.Error("请求失败: HTTP %d %s", statusCode, resp.Status)
		}
	}

	// 输出响应
	if output != "" {
		if err := os.WriteFile(output, respBody, 0644); err != nil {
			return err
		}
		if !showStatus {
			logger.Success("响应已保存到: %s", output)
		}
	} else {
		// 输出状态码和响应头
		fmt.Printf("HTTP/1.1 %d %s\n", resp.StatusCode, resp.Status)
		for k, v := range resp.Header {
			fmt.Printf("%s: %s\n", k, strings.Join(v, ", "))
		}
		fmt.Println()
		// 输出响应体
		fmt.Println(string(respBody))
	}

	return nil
}

// formatBytes 格式化字节数
func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
