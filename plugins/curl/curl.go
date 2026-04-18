package curl

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"opsxcli/internal/logger"
)

// Request 发送HTTP请求 (类curl)
func Request(url, method string, headers []string, data, output string, includeHeaders, verbose, followRedirect bool) error {
	// 如果URL没有协议前缀,自动添加https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// 如果有POST数据但method为GET，自动改为POST
	if data != "" && method == "GET" {
		method = "POST"
	}

	// 创建请求
	var body io.Reader
	if data != "" {
		body = strings.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	// 设置默认User-Agent(模仿真实curl)
	req.Header.Set("User-Agent", "curl/8.0.1")

	// 设置自定义请求头
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	// 如果有POST数据且没有Content-Type，自动设置
	if data != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// 显示详细信息
	if verbose {
		fmt.Printf("> %s %s HTTP/1.1\n", method, req.URL.RequestURI())
		fmt.Printf("> Host: %s\n", req.Host)
		for k, v := range req.Header {
			fmt.Printf("> %s: %s\n", k, strings.Join(v, ", "))
		}
		fmt.Println(">")
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 是否跟随重定向
	if !followRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// 发送请求
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 显示详细响应信息
	if verbose {
		duration := time.Since(startTime)
		fmt.Printf("< HTTP/%d.%d %d %s\n", resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status)
		for k, v := range resp.Header {
			fmt.Printf("< %s: %s\n", k, strings.Join(v, ", "))
		}
		fmt.Printf("<\n")
		fmt.Printf("* Request completed in %v\n", duration)
		fmt.Println()
	}

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 输出到文件
	if output != "" {
		if err := os.WriteFile(output, respBody, 0644); err != nil {
			return err
		}
		logger.Success("响应已保存到: %s", output)
		return nil
	}

	// 输出到标准输出
	if includeHeaders && !verbose {
		// 输出响应头（如果使用-i参数）
		fmt.Printf("HTTP/%d.%d %d %s\n", resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, resp.Status)
		for k, v := range resp.Header {
			fmt.Printf("%s: %s\n", k, strings.Join(v, ", "))
		}
		fmt.Println()
	}

	// 输出响应体（HEAD请求不输出body）
	if method != "HEAD" {
		fmt.Print(string(respBody))
		// 如果响应体不以换行结束，添加换行
		if len(respBody) > 0 && respBody[len(respBody)-1] != '\n' {
			fmt.Println()
		}
	}

	return nil
}
