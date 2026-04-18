package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Format 输出格式
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

var (
	format   Format = FormatText
	writer   io.Writer = os.Stdout
	quiet    bool       = false
)

// SetFormat 设置输出格式
func SetFormat(f Format) {
	format = f
}

// SetQuiet 设置静默模式
func SetQuiet(q bool) {
	quiet = q
}

// SetWriter 设置输出 writer（用于测试）
func SetWriter(w io.Writer) {
	writer = w
}

// IsJSON 是否为 JSON 模式
func IsJSON() bool {
	return format == FormatJSON
}

// IsQuiet 是否静默模式
func IsQuiet() bool {
	return quiet
}

// Print 将数据以配置的格式输出到 writer
// text 模式：直接打印字符串；json 模式：序列化为 JSON
func Print(data interface{}) {
	if quiet {
		return
	}
	switch format {
	case FormatJSON:
		enc := json.NewEncoder(writer)
		enc.SetIndent("", "  ")
		_ = enc.Encode(data)
	default:
		switch v := data.(type) {
		case string:
			fmt.Fprintln(writer, v)
		default:
			fmt.Fprintln(writer, v)
		}
	}
}

// JSON 将任意数据序列化为 JSON 并输出
func JSON(data interface{}) {
	if quiet {
		return
	}
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

// Result 通用命令结果结构（用于 JSON 输出）
type Result struct {
	Success   bool            `json:"success"`
	Message   string          `json:"message,omitempty"`
	Data      interface{}     `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	ExitCode  int             `json:"exit_code,omitempty"`
}
