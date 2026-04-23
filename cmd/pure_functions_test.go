package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"opsxcli/plugins/ssl"
)

// === getIPHelpText 测试 ===

func TestGetIPHelpText(t *testing.T) {
	text := getIPHelpText()

	if text == "" {
		t.Fatal("getIPHelpText() 返回空字符串")
	}

	// 验证帮助文本包含关键内容
	keywords := []string{
		"ip [ OPTIONS ] OBJECT",
		"link | address | route | help",
		"常用命令",
		"ip addr",
		"ip link",
		"ip route",
		"简写形式",
	}

	for _, kw := range keywords {
		if !strings.Contains(text, kw) {
			t.Errorf("帮助文本缺少关键内容: %q", kw)
		}
	}
}

// 确保每次调用都返回相同内容（纯函数无副作用）
func TestGetIPHelpText_Idempotent(t *testing.T) {
	first := getIPHelpText()
	second := getIPHelpText()
	if first != second {
		t.Error("getIPHelpText() 多次调用返回不同结果")
	}
}

// === handleIPCommand 测试 ===

func TestHandleIPCommand_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"help参数", []string{"help"}},
		{"短help", []string{"-h"}},
		{"长help", []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 捕获 stdout 输出
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := handleIPCommand(tt.args)

			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			os.Stdout = old

			if err != nil {
				t.Errorf("handleIPCommand(%v) 返回错误: %v", tt.args, err)
			}

			output := buf.String()
			if !strings.Contains(output, "ip [ OPTIONS ] OBJECT") {
				t.Errorf("帮助输出缺少关键内容, got: %s", output[:min(200, len(output))])
			}
		})
	}
}

func TestHandleIPCommand_UnknownObject(t *testing.T) {
	err := handleIPCommand([]string{"unknown_object"})
	if err == nil {
		t.Fatal("handleIPCommand 应该对未知对象返回错误")
	}
	if !strings.Contains(err.Error(), "未知") || !strings.Contains(err.Error(), "unknown_object") {
		t.Errorf("错误消息格式不对: %v", err)
	}
}

func TestHandleIPCommand_FilterOptions(t *testing.T) {
	// 测试选项参数被过滤后，遇到空 object 时显示 help
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 只有选项参数（以 - 开头），没有 object，最终 object=""，走 help 分支
	err := handleIPCommand([]string{"-s", "-4"})

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	if err != nil {
		t.Errorf("handleIPCommand 返回错误: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "ip [ OPTIONS ] OBJECT") {
		t.Errorf("选项过滤后应显示帮助, got: %s", output[:min(200, len(output))])
	}
}

// === outputJSONInfo 测试 ===

func TestOutputJSONInfo(t *testing.T) {
	now := time.Now()
	info := &ssl.CertificateInfo{
		Domain:             "example.com",
		Issuer:             "Test CA",
		Subject:            "example.com",
		NotBefore:          now,
		NotAfter:           now.Add(365 * 24 * time.Hour),
		DaysUntilExpiry:    365,
		IsExpired:          false,
		ChainLength:        2,
		SerialNumber:       "ABC123",
		SignatureAlgorithm: "SHA256-RSA",
		DNSNames:           []string{"example.com", "www.example.com"},
		Port:               443,
	}

	// 捕获 stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSONInfo(info)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputJSONInfo 返回错误: %v", err)
	}

	output := buf.String()

	// 验证是有效的 JSON
	var parsed ssl.CertificateInfo
	if jsonErr := json.Unmarshal([]byte(output), &parsed); jsonErr != nil {
		t.Fatalf("输出不是有效 JSON: %v, output: %s", jsonErr, output)
	}

	if parsed.Domain != "example.com" {
		t.Errorf("Domain 不匹配: got %q, want %q", parsed.Domain, "example.com")
	}
	if parsed.Issuer != "Test CA" {
		t.Errorf("Issuer 不匹配: got %q, want %q", parsed.Issuer, "Test CA")
	}
	if parsed.Port != 443 {
		t.Errorf("Port 不匹配: got %d, want %d", parsed.Port, 443)
	}
	if parsed.DaysUntilExpiry != 365 {
		t.Errorf("DaysUntilExpiry 不匹配: got %d, want %d", parsed.DaysUntilExpiry, 365)
	}
	if len(parsed.DNSNames) != 2 {
		t.Errorf("DNSNames 长度不匹配: got %d, want 2", len(parsed.DNSNames))
	}
}

func TestOutputJSONInfo_NilInfo(t *testing.T) {
	// 输出 nil CertificateInfo，json.MarshalIndent 会输出 "null"
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSONInfo(nil)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputJSONInfo(nil) 返回错误: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "null" {
		t.Errorf("outputJSONInfo(nil) 应输出 'null', got: %q", buf.String())
	}
}

// === outputJSONChain 测试 ===

func TestOutputJSONChain(t *testing.T) {
	now := time.Now()
	chain := []*ssl.CertificateInfo{
		{
			Domain:          "example.com",
			Issuer:          "Intermediate CA",
			NotBefore:       now,
			NotAfter:        now.Add(365 * 24 * time.Hour),
			DaysUntilExpiry: 365,
			IsExpired:       false,
		},
		{
			Domain:          "Intermediate CA",
			Issuer:          "Root CA",
			NotBefore:       now,
			NotAfter:        now.Add(3650 * 24 * time.Hour),
			DaysUntilExpiry: 3650,
			IsExpired:       false,
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSONChain(chain)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputJSONChain 返回错误: %v", err)
	}

	output := buf.String()
	var parsed []*ssl.CertificateInfo
	if jsonErr := json.Unmarshal([]byte(output), &parsed); jsonErr != nil {
		t.Fatalf("输出不是有效 JSON: %v, output: %s", jsonErr, output)
	}

	if len(parsed) != 2 {
		t.Fatalf("证书链长度不匹配: got %d, want 2", len(parsed))
	}
	if parsed[0].Domain != "example.com" {
		t.Errorf("第一个证书 Domain 不匹配: got %q", parsed[0].Domain)
	}
	if parsed[1].Domain != "Intermediate CA" {
		t.Errorf("第二个证书 Domain 不匹配: got %q", parsed[1].Domain)
	}
}

func TestOutputJSONChain_Empty(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSONChain([]*ssl.CertificateInfo{})

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputJSONChain([]) 返回错误: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Errorf("空链应输出 '[]', got: %q", buf.String())
	}
}

func TestOutputJSONChain_Nil(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSONChain(nil)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	if err != nil {
		t.Fatalf("outputJSONChain(nil) 返回错误: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "null" {
		t.Errorf("nil 链应输出 'null', got: %q", buf.String())
	}
}

// === showAgentHelp 测试 ===

func TestShowAgentHelp(t *testing.T) {
	// showAgentHelp 不使用 ag 参数，传 nil 即可
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	showAgentHelp(nil)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()

	// 验证帮助信息包含关键内容
	expectedKeywords := []string{
		"Agent 帮助",
		"/help",
		"/exit",
		"/quit",
		"/tasks",
		"用法示例",
		"opsxcli agent",
		"交互命令",
	}

	for _, kw := range expectedKeywords {
		if !strings.Contains(output, kw) {
			t.Errorf("Agent 帮助缺少关键内容: %q", kw)
		}
	}
}

// === printSearchResults 测试 ===

func TestPrintSearchResults(t *testing.T) {
	results := []searchResult{
		{
			Name:        "redis",
			Category:    "数据库",
			Description: "Redis 客户端工具",
			Aliases:     []string{"rds"},
		},
		{
			Name:        "mysql",
			Category:    "数据库",
			Description: "MySQL 客户端工具",
			Aliases:     nil,
		},
		{
			Name:        "ping",
			Category:    "网络",
			Description: "网络连通性测试",
			Aliases:     []string{"p"},
		},
	}

	keywords := []string{"测试", "工具"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSearchResults(results, keywords)

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()

	// 验证输出包含命令名
	for _, res := range results {
		if !strings.Contains(output, res.Name) {
			t.Errorf("搜索结果缺少命令名: %q", res.Name)
		}
		if !strings.Contains(output, res.Description) {
			t.Errorf("搜索结果缺少描述: %q", res.Description)
		}
	}

	// 验证分类标签
	if !strings.Contains(output, "[数据库]") {
		t.Error("搜索结果缺少分类 [数据库]")
	}
	if !strings.Contains(output, "[网络]") {
		t.Error("搜索结果缺少分类 [网络]")
	}

	// 验证关键词显示
	if !strings.Contains(output, "测试 + 工具") {
		t.Error("搜索结果应显示关键词")
	}

	// 验证匹配数量
	if !strings.Contains(output, "3 个匹配命令") {
		t.Error("搜索结果应显示匹配数量")
	}
}

func TestPrintSearchResults_SingleResult(t *testing.T) {
	results := []searchResult{
		{
			Name:        "ssl",
			Category:    "网络",
			Description: "SSL证书检查工具",
			Aliases:     []string{},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSearchResults(results, []string{"ssl"})

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	output := buf.String()
	if !strings.Contains(output, "1 个匹配命令") {
		t.Error("应显示 1 个匹配命令")
	}
	if !strings.Contains(output, "ssl") {
		t.Error("应包含命令名 ssl")
	}
}

func TestPrintSearchResults_EmptyAliases(t *testing.T) {
	results := []searchResult{
		{
			Name:        "test",
			Category:    "工具",
			Description: "测试命令",
			Aliases:     nil,
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printSearchResults(results, []string{"test"})

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old

	// 应该正常输出，不会 panic
	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Error("应包含命令名 test")
	}
}

// === searchCommands 测试 ===

func TestSearchCommands(t *testing.T) {
	// init() 已注册了一些命令（如 ip, ifconfig, route 等）
	tests := []struct {
		name     string
		keywords []string
		// 不验证具体结果（依赖 init 注册），只验证不会 panic 且返回正确类型
	}{
		{"搜索IP", []string{"ip"}},
		{"搜索网络", []string{"网络"}},
		{"多关键词", []string{"系统", "网络"}},
		{"无匹配", []string{"zzz_no_match_zzz"}},
		{"空关键词", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 不应 panic
			results := searchCommands(tt.keywords)
			// 验证排序：结果按 Category 升序、同 Category 按 Name 升序
			for i := 1; i < len(results); i++ {
				prev := results[i-1]
				curr := results[i]
				if prev.Category > curr.Category {
					t.Errorf("结果未按 Category 排序: %q > %q", prev.Category, curr.Category)
				}
				if prev.Category == curr.Category && prev.Name > curr.Name {
					t.Errorf("同分类内未按 Name 排序: %q > %q", prev.Name, curr.Name)
				}
			}
		})
	}
}

// helper
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
