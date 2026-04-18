package wget

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"opsxcli/internal/logger"
)

// Download 下载文件
func Download(url, output string, continueDownload bool) error {
	// 如果URL没有协议前缀,自动添加https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// 创建HTTP请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	// 设置默认User-Agent(模仿真实wget)
	req.Header.Set("User-Agent", "Wget/1.21.3")

	// 如果启用断点续传，检查本地文件
	var file *os.File
	var fileSize int64
	if continueDownload && output != "" {
		if info, err := os.Stat(output); err == nil {
			fileSize = info.Size()
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", fileSize))
			file, err = os.OpenFile(output, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			logger.Info("继续下载: %s (已下载: %d bytes)", output, fileSize)
		}
	}

	// 如果文件未打开，创建新文件
	if file == nil {
		if output == "" {
			// 从URL提取文件名
			output = extractFilename(url)
		}
		file, err = os.Create(output)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		err := fmt.Errorf("下载失败: %s", resp.Status)
		logger.Error("Wget下载失败 %s: %v", url, err)
		return err
	}

	// 获取文件大小
	totalSize := resp.ContentLength
	if totalSize > 0 {
		totalSize += fileSize
	}

	// 复制数据
	var written int64
	if totalSize > 0 {
		written, err = copyWithProgress(file, resp.Body, fileSize, totalSize)
	} else {
		written, err = io.Copy(file, resp.Body)
	}

	if err != nil {
		return err
	}

	logger.Success("下载完成: %s (%d bytes)", output, written)
	return nil
}

// 辅助函数
func extractFilename(url string) string {
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	if filename == "" {
		return "index.html"
	}
	// 移除查询参数
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}
	return filename
}

func copyWithProgress(dst io.Writer, src io.Reader, start, total int64) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	written = start
	var err error

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}

		// 显示进度
		if total > 0 {
			percent := float64(written) / float64(total) * 100
			fmt.Printf("\r进度: %.1f%% (%d/%d bytes)", percent, written, total)
		}
	}

	fmt.Println() // 换行
	return written, err
}
