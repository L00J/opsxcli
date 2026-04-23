package ssh

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// parseTarget 测试
// ============================================================================

// TestParseTarget 使用表驱动测试覆盖 parseTarget 的各种输入情况
func TestParseTarget(t *testing.T) {
	tests := []struct {
		name     string // 测试用例描述
		input    string // 输入的目标字符串
		wantUser string // 期望返回的用户名
		wantHost string // 期望返回的主机地址
		wantErr  bool   // 是否期望返回错误
	}{
		// --- 基本功能测试 ---
		{
			name:     "仅主机名-默认用户为root",
			input:    "192.168.1.1",
			wantUser: "root",
			wantHost: "192.168.1.1",
			wantErr:  false,
		},
		{
			name:     "标准user@host格式",
			input:    "admin@192.168.1.1",
			wantUser: "admin",
			wantHost: "192.168.1.1",
			wantErr:  false,
		},
		{
			name:     "用户名为空@host",
			input:    "@example.com",
			wantUser: "",
			wantHost: "example.com",
			wantErr:  false,
		},

		// --- 边界条件和错误情况 ---
		{
			// strings.Split("", "@") 返回 [""]（长度为1），走 len==1 分支
			name:     "空字符串-Split返回单元素切片,默认root用户",
			input:    "",
			wantUser: "root",
			wantHost: "",
			wantErr:  false,
		},
		{
			name:    "多个@符号-返回错误",
			input:   "a@b@c",
			wantErr: true,
		},
		{
			name:    "三个@符号-返回错误",
			input:   "a@b@c@d",
			wantErr: true,
		},
		{
			name:     "用户名包含特殊字符",
			input:    "user.name@host.com",
			wantUser: "user.name",
			wantHost: "host.com",
			wantErr:  false,
		},
		{
			name:     "主机为IP带端口(不含@)",
			input:    "10.0.0.1:22",
			wantUser: "root",
			wantHost: "10.0.0.1:22",
			wantErr:  false,
		},
		{
			name:     "主机为域名",
			input:    "deploy@api.example.com",
			wantUser: "deploy",
			wantHost: "api.example.com",
			wantErr:  false,
		},
		{
			name:     "仅@符号-用户和主机均为空",
			input:    "@",
			wantUser: "",
			wantHost: "",
			wantErr:  false,
		},
		{
			name:     "主机为localhost",
			input:    "root@localhost",
			wantUser: "root",
			wantHost: "localhost",
			wantErr:  false,
		},
		{
			name:     "用户名中包含连字符",
			input:    "my-user@server.net",
			wantUser: "my-user",
			wantHost: "server.net",
			wantErr:  false,
		},
		{
			name:     "用户名中包含下划线",
			input:    "my_user@server.net",
			wantUser: "my_user",
			wantHost: "server.net",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, gotHost, gotErr := parseTarget(tt.input)
			if tt.wantErr {
				assert.Error(t, gotErr, "输入 %q 应该返回错误", tt.input)
			} else {
				assert.NoError(t, gotErr, "输入 %q 不应该返回错误", tt.input)
				assert.Equal(t, tt.wantUser, gotUser, "用户名不匹配")
				assert.Equal(t, tt.wantHost, gotHost, "主机地址不匹配")
			}
		})
	}
}

// ============================================================================
// parseRemotePath 测试
// ============================================================================

// TestParseRemotePath 使用表驱动测试覆盖 parseRemotePath 的各种输入情况
func TestParseRemotePath(t *testing.T) {
	tests := []struct {
		name     string // 测试用例描述
		input    string // 输入的远程路径字符串
		wantUser string // 期望返回的用户名
		wantHost string // 期望返回的主机地址
		wantPath string // 期望返回的路径
		wantErr  bool   // 是否期望返回错误
	}{
		// --- 基本功能测试 ---
		{
			name:     "标准user@host:/path格式",
			input:    "admin@192.168.1.1:/etc/hosts",
			wantUser: "admin",
			wantHost: "192.168.1.1",
			wantPath: "/etc/hosts",
			wantErr:  false,
		},
		{
			name:     "仅host:/path格式-默认用户为root",
			input:    "192.168.1.1:/var/log/syslog",
			wantUser: "root",
			wantHost: "192.168.1.1",
			wantPath: "/var/log/syslog",
			wantErr:  false,
		},
		{
			name:     "路径为根目录",
			input:    "root@server:/",
			wantUser: "root",
			wantHost: "server",
			wantPath: "/",
			wantErr:  false,
		},
		{
			name:     "路径包含多级目录",
			input:    "deploy@app.example.com:/opt/app/data/logs",
			wantUser: "deploy",
			wantHost: "app.example.com",
			wantPath: "/opt/app/data/logs",
			wantErr:  false,
		},

		// --- 边界条件和错误情况 ---
		{
			name:    "空字符串-缺少冒号返回错误",
			input:   "",
			wantErr: true,
		},
		{
			name:    "缺少冒号-返回错误",
			input:   "admin@192.168.1.1",
			wantErr: true,
		},
		{
			name:    "多个@符号-返回错误",
			input:   "a@b@c:/path",
			wantErr: true,
		},
		{
			name:     "路径为空字符串",
			input:    "admin@192.168.1.1:",
			wantUser: "admin",
			wantHost: "192.168.1.1",
			wantPath: "",
			wantErr:  false,
		},
		{
			name:     "相对路径",
			input:    "user@host:relative/path",
			wantUser: "user",
			wantHost: "host",
			wantPath: "relative/path",
			wantErr:  false,
		},
		{
			name:     "使用LastIndex处理IPv6含冒号的情况",
			input:    "root@[::1]:/tmp/file",
			wantUser: "root",
			wantHost: "[::1]", // LastIndex 找到最后一个冒号，所以 host 为 [::1]
			wantPath: "/tmp/file",
			wantErr:  false,
		},
		{
			name:     "路径包含中文",
			input:    "admin@server:/data/日志文件",
			wantUser: "admin",
			wantHost: "server",
			wantPath: "/data/日志文件",
			wantErr:  false,
		},
		{
			name:     "路径包含空格",
			input:    "admin@server:/path/with spaces/file",
			wantUser: "admin",
			wantHost: "server",
			wantPath: "/path/with spaces/file",
			wantErr:  false,
		},
		{
			name:     "仅冒号-路径和host@user均为空",
			input:    ":",
			wantUser: "root",
			wantHost: "",
			wantPath: "",
			wantErr:  false,
		},
		{
			name:    "仅host无用户无路径-缺少冒号返回错误",
			input:   "server",
			wantErr: true,
		},
		{
			name:     "主机名含端口-LastIndex取最后一个冒号",
			input:    "admin@host:22:/tmp",
			wantUser: "admin",
			wantHost: "host:22", // LastIndex 取最后一个冒号之前的部分
			wantPath: "/tmp",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, gotHost, gotPath, gotErr := parseRemotePath(tt.input)
			if tt.wantErr {
				assert.Error(t, gotErr, "输入 %q 应该返回错误", tt.input)
			} else {
				assert.NoError(t, gotErr, "输入 %q 不应该返回错误", tt.input)
				assert.Equal(t, tt.wantUser, gotUser, "用户名不匹配")
				assert.Equal(t, tt.wantHost, gotHost, "主机地址不匹配")
				assert.Equal(t, tt.wantPath, gotPath, "路径不匹配")
			}
		})
	}
}

// ============================================================================
// parseAddr 测试
// ============================================================================

// TestParseAddr 使用表驱动测试覆盖 parseAddr 的各种输入情况
func TestParseAddr(t *testing.T) {
	tests := []struct {
		name     string // 测试用例描述
		input    string // 输入的地址字符串
		wantHost string // 期望返回的主机地址
		wantPort int    // 期望返回的端口号
		wantErr  bool   // 是否期望返回错误
	}{
		// --- 基本功能测试 ---
		{
			name:     "标准host:port格式",
			input:    "127.0.0.1:8080",
			wantHost: "127.0.0.1",
			wantPort: 8080,
			wantErr:  false,
		},
		{
			name:     "localhost带端口",
			input:    "localhost:3000",
			wantHost: "localhost",
			wantPort: 3000,
			wantErr:  false,
		},
		{
			name:     "0.0.0.0绑定所有接口",
			input:    "0.0.0.0:9090",
			wantHost: "0.0.0.0",
			wantPort: 9090,
			wantErr:  false,
		},
		{
			name:     "端口为0",
			input:    "0.0.0.0:0",
			wantHost: "0.0.0.0",
			wantPort: 0,
			wantErr:  false,
		},
		{
			name:     "端口为1(最小合法端口)",
			input:    "host:1",
			wantHost: "host",
			wantPort: 1,
			wantErr:  false,
		},
		{
			name:     "端口为65535(最大合法端口)",
			input:    "host:65535",
			wantHost: "host",
			wantPort: 65535,
			wantErr:  false,
		},
		{
			name:     "空主机名带端口",
			input:    ":8080",
			wantHost: "",
			wantPort: 8080,
			wantErr:  false,
		},

		// --- 错误情况 ---
		{
			name:    "空字符串-返回错误",
			input:   "",
			wantErr: true,
		},
		{
			name:    "缺少冒号-返回错误",
			input:   "192.168.1.1",
			wantErr: true,
		},
		{
			name:    "多个冒号-返回错误(IPv6)",
			input:   "::1:8080",
			wantErr: true,
		},
		{
			name:    "端口不是数字-返回错误",
			input:   "host:abc",
			wantErr: true,
		},
		{
			name:    "端口包含空格-返回错误",
			input:   "host:80 80",
			wantErr: true,
		},
		{
			name:    "端口为浮点数-返回错误",
			input:   "host:80.5",
			wantErr: true,
		},
		{
			// strconv.Atoi("-1") 成功解析为 -1，parseAddr 不做范围校验
			name:     "端口为负数-strconv.Atoi可解析但非合法端口",
			input:    "host:-1",
			wantHost: "host",
			wantPort: -1,
			wantErr:  false,
		},
		{
			// strconv.Atoi("+8080") 成功解析为 8080
			name:     "端口带加号前缀-strconv.Atoi正常解析",
			input:    "host:+8080",
			wantHost: "host",
			wantPort: 8080,
			wantErr:  false,
		},
		{
			name:    "三个冒号-返回错误",
			input:   "a:b:c",
			wantErr: true,
		},
		{
			name:    "仅有冒号-返回错误",
			input:   ":",
			wantErr: true,
		},
		{
			name:    "端口为十六进制-返回错误",
			input:   "host:0x50",
			wantErr: true,
		},
		{
			name:     "端口带前导零(非标准但strconv.Atoi可解析)-验证行为",
			input:    "host:08080",
			wantHost: "host",
			wantPort: 8080,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHost, gotPort, gotErr := parseAddr(tt.input)
			if tt.wantErr {
				assert.Error(t, gotErr, "输入 %q 应该返回错误", tt.input)
			} else {
				assert.NoError(t, gotErr, "输入 %q 不应该返回错误", tt.input)
				assert.Equal(t, tt.wantHost, gotHost, "主机地址不匹配")
				assert.Equal(t, tt.wantPort, gotPort, "端口号不匹配")
			}
		})
	}
}

// ============================================================================
// 集成交叉验证测试
// ============================================================================

// TestParseTargetAndRemotePathConsistency 验证 parseRemotePath 正确委托给 parseTarget
func TestParseTargetAndRemotePathConsistency(t *testing.T) {
	// parseRemotePath 应该将 user@host 部分正确委托给 parseTarget
	input := "admin@web01.example.com:/var/www/html"
	user1, host1, path1, err1 := parseRemotePath(input)
	assert.NoError(t, err1)

	user2, host2, err2 := parseTarget("admin@web01.example.com")
	assert.NoError(t, err2)

	// 两者的用户名和主机解析结果应一致
	assert.Equal(t, user2, user1, "parseRemotePath 和 parseTarget 的用户名结果应一致")
	assert.Equal(t, host2, host1, "parseRemotePath 和 parseTarget 的主机结果应一致")
	assert.Equal(t, "/var/www/html", path1, "路径应正确解析")
	_ = host1 + host2 // 消除未使用变量警告
	_ = user1 + user2
}

// TestParseAddrPortOutOfRange 验证超出范围的端口值
// 注意：parseAddr 不做范围校验，仅检查是否为合法整数
// 如果端口号 > 65535，函数不会返回错误（但实际使用中不应如此）
func TestParseAddrPortOutOfRange(t *testing.T) {
	// 端口 65536 超出 TCP/UDP 合法范围，但 parseAddr 不做范围校验
	host, port, err := parseAddr("host:65536")
	assert.NoError(t, err, "parseAddr 不做端口范围校验，不应该返回错误")
	assert.Equal(t, "host", host)
	assert.Equal(t, 65536, port)

	// 极大端口号
	host, port, err = parseAddr("host:99999")
	assert.NoError(t, err)
	assert.Equal(t, 99999, port)
	_ = host

	// 负端口（strconv.Atoi 无法解析带负号的非数字错误情况已覆盖）
	// 这里测试一个极大正数
	_, port, err = parseAddr("host:" + strconv.Itoa(1<<20))
	assert.NoError(t, err)
	assert.Equal(t, 1<<20, port)
}

// ============================================================================
// 基准测试
// ============================================================================

// BenchmarkParseTarget 基准测试 parseTarget 函数的性能
func BenchmarkParseTarget(b *testing.B) {
	// 预定义测试输入，避免在循环中分配
	inputs := []string{
		"root@192.168.1.1",
		"192.168.1.1",
		"admin@api.example.com",
	}

	b.ResetTimer() // 重置计时器，排除初始化开销
	for i := 0; i < b.N; i++ {
		_, _, _ = parseTarget(inputs[i%len(inputs)])
	}
}

// BenchmarkParseTargetOnlyHost 基准测试仅主机名输入的场景
func BenchmarkParseTargetOnlyHost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseTarget("192.168.1.1")
	}
}

// BenchmarkParseTargetUserAndHost 基准测试标准 user@host 输入的场景
func BenchmarkParseTargetUserAndHost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseTarget("admin@192.168.1.1")
	}
}

// BenchmarkParseRemotePath 基准测试 parseRemotePath 函数的性能
func BenchmarkParseRemotePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseRemotePath("admin@192.168.1.1:/var/log/syslog")
	}
}

// BenchmarkParseAddr 基准测试 parseAddr 函数的性能
func BenchmarkParseAddr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseAddr("127.0.0.1:8080")
	}
}
