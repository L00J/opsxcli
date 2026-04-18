package busybox

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// DdOptions dd 命令选项
type DdOptions struct {
	If         string // input file
	Of         string // output file
	Bs         int64  // block size
	Count      int64  // number of blocks to copy
	Skip       int64  // skip N blocks at start of input
	Seek       int64  // skip N blocks at start of output
	Conv       string // conversion options (如: notrunc, sync)
	Status     string // status level (progress, noxfer, none)
	IFlag      string // input flags
	OFlag      string // output flags
}

// Dd 实现 dd 命令
func Dd(opts DdOptions) error {
	// 默认值
	if opts.Bs == 0 {
		opts.Bs = 512 // 默认块大小 512 字节
	}
	if opts.Status == "" {
		opts.Status = "progress"
	}

	// 打开输入
	var input io.Reader
	if opts.If == "" || opts.If == "-" {
		input = os.Stdin
		opts.If = "stdin"
	} else {
		f, err := os.Open(opts.If)
		if err != nil {
			return fmt.Errorf("打开输入文件失败: %v", err)
		}
		defer f.Close()
		input = f

		// 跳过输入块
		if opts.Skip > 0 {
			skipBytes := opts.Skip * opts.Bs
			if _, err := f.Seek(skipBytes, io.SeekStart); err != nil {
				return fmt.Errorf("跳过输入块失败: %v", err)
			}
		}
	}

	// 打开输出
	var output io.Writer
	var outputFile *os.File
	if opts.Of == "" || opts.Of == "-" {
		output = os.Stdout
		opts.Of = "stdout"
	} else {
		// 检查 conv 选项
		notrunc := strings.Contains(opts.Conv, "notrunc")

		var err error
		var flags int
		if notrunc {
			// notrunc: 不截断输出文件
			flags = os.O_RDWR | os.O_CREATE
		} else {
			// 默认: 截断输出文件
			flags = os.O_RDWR | os.O_CREATE | os.O_TRUNC
		}

		outputFile, err = os.OpenFile(opts.Of, flags, 0644)
		if err != nil {
			return fmt.Errorf("打开输出文件失败: %v", err)
		}
		defer outputFile.Close()
		output = outputFile

		// 跳过输出块
		if opts.Seek > 0 {
			seekBytes := opts.Seek * opts.Bs
			if _, err := outputFile.Seek(seekBytes, io.SeekStart); err != nil {
				return fmt.Errorf("跳过输出块失败: %v", err)
			}
		}
	}

	// 执行复制
	var totalBytes int64
	var blocksRead int64
	var blocksWritten int64
	startTime := time.Now()

	buf := make([]byte, opts.Bs)
	showProgress := opts.Status == "progress" && opts.Of != "stdout"

	for {
		// 检查是否达到 count 限制
		if opts.Count > 0 && blocksRead >= opts.Count {
			break
		}

		// 读取一个块
		n, err := io.ReadFull(input, buf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if n > 0 {
					// 最后一个不完整的块
					blocksRead++
					if _, wErr := output.Write(buf[:n]); wErr != nil {
						return fmt.Errorf("写入失败: %v", wErr)
					}
					totalBytes += int64(n)
					blocksWritten++
				}
				break
			}
			return fmt.Errorf("读取失败: %v", err)
		}

		blocksRead++

		// 写入块
		if _, err := output.Write(buf[:n]); err != nil {
			return fmt.Errorf("写入失败: %v", err)
		}
		totalBytes += int64(n)
		blocksWritten++

		// 显示进度
		if showProgress && blocksWritten%100 == 0 {
			elapsed := time.Since(startTime).Seconds()
			speed := float64(totalBytes) / elapsed / 1024 / 1024 // MB/s
			fmt.Fprintf(os.Stderr, "\r已复制 %d 字节 (%.2f MB, %.2f MB/s)",
				totalBytes, float64(totalBytes)/1024/1024, speed)
		}
	}

	// 清除进度行
	if showProgress {
		fmt.Fprint(os.Stderr, "\r\033[K")
	}

	// 显示统计信息 (除非 status=none)
	if opts.Status != "none" {
		elapsed := time.Since(startTime).Seconds()
		speed := float64(totalBytes) / elapsed

		fmt.Fprintf(os.Stderr, "%d+0 records in\n", blocksRead)
		fmt.Fprintf(os.Stderr, "%d+0 records out\n", blocksWritten)
		fmt.Fprintf(os.Stderr, "%d bytes (%s) copied, %.4f s, %.1f MB/s\n",
			totalBytes,
			formatBytes(totalBytes),
			elapsed,
			speed/1024/1024)
	}

	return nil
}

// formatBytes 格式化字节数为人类可读格式
func formatBytes(bytes int64) string {
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

// ParseDdArgs 解析 dd 命令行参数
func ParseDdArgs(args []string) (DdOptions, error) {
	opts := DdOptions{}

	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return opts, fmt.Errorf("无效的参数格式: %s (应为 key=value)", arg)
		}

		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "if":
			opts.If = value
		case "of":
			opts.Of = value
		case "bs":
			size, err := parseSize(value)
			if err != nil {
				return opts, fmt.Errorf("无效的 bs 值: %v", err)
			}
			opts.Bs = size
		case "count":
			count, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return opts, fmt.Errorf("无效的 count 值: %v", err)
			}
			opts.Count = count
		case "skip":
			skip, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return opts, fmt.Errorf("无效的 skip 值: %v", err)
			}
			opts.Skip = skip
		case "seek":
			seek, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return opts, fmt.Errorf("无效的 seek 值: %v", err)
			}
			opts.Seek = seek
		case "conv":
			opts.Conv = value
		case "status":
			if value != "progress" && value != "noxfer" && value != "none" {
				return opts, fmt.Errorf("无效的 status 值: %s (应为 progress, noxfer 或 none)", value)
			}
			opts.Status = value
		case "iflag":
			opts.IFlag = value
		case "oflag":
			opts.OFlag = value
		default:
			return opts, fmt.Errorf("未知参数: %s", key)
		}
	}

	return opts, nil
}

// parseSize 解析大小字符串 (支持 K, M, G 后缀)
func parseSize(s string) (int64, error) {
	s = strings.ToUpper(s)
	multiplier := int64(1)

	if len(s) > 0 {
		lastChar := s[len(s)-1]
		switch lastChar {
		case 'K':
			multiplier = 1024
			s = s[:len(s)-1]
		case 'M':
			multiplier = 1024 * 1024
			s = s[:len(s)-1]
		case 'G':
			multiplier = 1024 * 1024 * 1024
			s = s[:len(s)-1]
		case 'B':
			// 移除 B 后缀
			s = s[:len(s)-1]
			// 检查是否有单位前缀
			if len(s) > 0 {
				lastChar = s[len(s)-1]
				switch lastChar {
				case 'K':
					multiplier = 1024
					s = s[:len(s)-1]
				case 'M':
					multiplier = 1024 * 1024
					s = s[:len(s)-1]
				case 'G':
					multiplier = 1024 * 1024 * 1024
					s = s[:len(s)-1]
				}
			}
		}
	}

	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}

	return num * multiplier, nil
}
