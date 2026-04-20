package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// === shortDigest 测试 ===

func TestShortDigest(t *testing.T) {
	tests := []struct {
		name    string
		digest  string
		want    string
	}{
		{"标准sha256摘要", "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "abcdef123456"},
		{"短摘要", "sha256:12345678901234567890", "123456789012"},
		{"无算法前缀", "abcdef1234567890abcdef1234567890", "abcdef123456"},
		{"空算法前缀", ":12345678901234567890", "123456789012"},
		{"恰好12字符", "sha256:123456789012", "123456789012"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shortDigest(tt.digest)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === formatSize 测试 ===

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{"零字节", 0, "0 B"},
		{"1字节", 1, "1 B"},
		{"512字节", 512, "512 B"},
		{"1KB", 1024, "1.00 KB"},
		{"1.5KB", 1536, "1.50 KB"},
		{"1MB", 1024 * 1024, "1.00 MB"},
		{"1.5MB", 1024 * 1024 * 3 / 2, "1.50 MB"},
		{"512MB", 512 * 1024 * 1024, "512.00 MB"},
		{"1GB", 1024 * 1024 * 1024, "1.00 GB"},
		{"2.5GB", int64(2.5 * 1024 * 1024 * 1024), "2.50 GB"},
		{"接近1KB", 1023, "1023 B"},
		{"接近1MB", 1024*1024 - 1, "1024.00 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.size)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === buildProxyImageName 测试 ===

func TestBuildProxyImageName(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		image    string
		want     string
	}{
		// 官方镜像 - 添加 library/ 前缀
		{"官方镜像nginx", "registry.example.com", "nginx", "registry.example.com/library/nginx"},
		{"官方镜像带标签", "registry.example.com", "nginx:latest", "registry.example.com/library/nginx:latest"},
		{"官方镜像redis", "registry.example.com", "redis:7", "registry.example.com/library/redis:7"},

		// 非官方镜像 - 已有路径
		{"用户镜像带路径", "registry.example.com", "myuser/myapp", "registry.example.com/myuser/myapp"},
		{"用户镜像带标签", "registry.example.com", "myuser/myapp:v1.0", "registry.example.com/myuser/myapp:v1.0"},

		// 已包含registry的镜像 - 直接返回
		{"已含registry的镜像", "registry.example.com", "docker.io/library/nginx", "docker.io/library/nginx"},
		{"已含完整域名", "registry.example.com", "ghcr.io/org/image:v1", "ghcr.io/org/image:v1"},

		// 边界情况 - 空registry会导致前缀为 "/"
		{"空registry官方镜像", "", "nginx", "/library/nginx"},
		{"空registry用户镜像", "", "user/app", "/user/app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildProxyImageName(tt.registry, tt.image)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === parseImageAndTag 测试 ===

func TestParseImageAndTag(t *testing.T) {
	tests := []struct {
		name      string
		image     string
		wantName  string
		wantTag   string
	}{
		// 标准格式
		{"镜像名和标签", "nginx:latest", "nginx", "latest"},
		{"镜像名和版本标签", "nginx:1.25", "nginx", "1.25"},
		{"用户镜像和标签", "myuser/myapp:v1.0", "myuser/myapp", "v1.0"},

		// 无标签 - 默认latest
		{"只有镜像名", "nginx", "nginx", "latest"},
		{"只有用户镜像", "myuser/myapp", "myuser/myapp", "latest"},

		// 带协议前缀
		{"https前缀", "https://registry.example.com/nginx:v1", "registry.example.com/nginx", "v1"},
		{"http前缀", "http://registry.example.com/nginx:v2", "registry.example.com/nginx", "v2"},

		// 带digest
		{"带sha256 digest", "nginx@sha256:abc123", "nginx", "sha256:abc123"},

		// 复杂情况
		{"带端口的镜像仓库", "registry.example.com:5000/myapp:v1", "registry.example.com", "5000/myapp"},
		{"带路径和标签", "registry.example.com/org/image:v2", "registry.example.com/org/image", "v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotTag := parseImageAndTag(tt.image)
			assert.Equal(t, tt.wantName, gotName, "镜像名不匹配")
			assert.Equal(t, tt.wantTag, gotTag, "标签不匹配")
		})
	}
}

// === parseAuthParams 测试 ===

func TestParseAuthParams(t *testing.T) {
	tests := []struct {
		name    string
		authStr string
		want    map[string]string
	}{
		{"简单键值对", "user=admin", map[string]string{"user": "admin"}},
		{"多个键值对", "user=admin,pass=secret", map[string]string{"user": "admin", "pass": "secret"}},
		{"带引号的值", `user="admin",pass="secret"`, map[string]string{"user": "admin", "pass": "secret"}},
		{"带空格", "user = admin , pass = secret", map[string]string{"user": "admin", "pass": "secret"}},
		{"空字符串", "", map[string]string{}},
		{"单个无值键", "keyonly", map[string]string{}},
		{"混合有效和无效", "valid=ok,badkey,another=yes", map[string]string{"valid": "ok", "another": "yes"}},
		{"值含等号", "token=abc=def", map[string]string{"token": "abc=def"}},
		{"带单侧引号", `user="admin`, map[string]string{"user": "admin"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAuthParams(tt.authStr)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === SelectFastestRegistries 测试 ===

func TestSelectFastestRegistries(t *testing.T) {
	tests := []struct {
		name    string
		results []RegistrySpeedTest
		count   int
		want    []string
	}{
		{
			"正常选择最快2个",
			[]RegistrySpeedTest{
				{Registry: "reg1", Latency: 50 * time.Millisecond, Success: true},
				{Registry: "reg2", Latency: 100 * time.Millisecond, Success: true},
				{Registry: "reg3", Latency: 150 * time.Millisecond, Success: true},
			},
			2,
			[]string{"reg1", "reg2"},
		},
		{
			"跳过失败的镜像源",
			[]RegistrySpeedTest{
				{Registry: "reg1", Latency: 50 * time.Millisecond, Success: true},
				{Registry: "reg2", Latency: 0, Success: false, Error: assert.AnError},
				{Registry: "reg3", Latency: 100 * time.Millisecond, Success: true},
			},
			3,
			[]string{"reg1", "reg3"},
		},
		{
			"全部失败",
			[]RegistrySpeedTest{
				{Registry: "reg1", Success: false, Error: assert.AnError},
				{Registry: "reg2", Success: false, Error: assert.AnError},
			},
			2,
			[]string(nil),
		},
		{
			"空结果",
			[]RegistrySpeedTest{},
			3,
			[]string(nil),
		},
		{
			"count为0选择全部成功的",
			[]RegistrySpeedTest{
				{Registry: "reg1", Latency: 50 * time.Millisecond, Success: true},
				{Registry: "reg2", Success: false, Error: assert.AnError},
			},
			0,
			[]string{"reg1"},
		},
		{
			"count超过结果数",
			[]RegistrySpeedTest{
				{Registry: "reg1", Latency: 50 * time.Millisecond, Success: true},
			},
			10,
			[]string{"reg1"},
		},
		{
			"count为负数选择全部",
			[]RegistrySpeedTest{
				{Registry: "reg1", Latency: 50 * time.Millisecond, Success: true},
				{Registry: "reg2", Latency: 80 * time.Millisecond, Success: true},
			},
			-1,
			[]string{"reg1", "reg2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectFastestRegistries(tt.results, tt.count)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === ConvertChunksToResumeInfo 测试 ===

func TestConvertChunksToResumeInfo(t *testing.T) {
	tests := []struct {
		name   string
		chunks []ChunkInfo
		want   []ChunkResumeInfo
	}{
		{
			"正常转换",
			[]ChunkInfo{
				{Index: 0, Start: 0, End: 1023, Size: 1024, Completed: true, Registry: "reg1"},
				{Index: 1, Start: 1024, End: 2047, Size: 1024, Completed: false, Registry: "reg2"},
			},
			[]ChunkResumeInfo{
				{Index: 0, Start: 0, End: 1023, Size: 1024, Completed: true, Registry: "reg1"},
				{Index: 1, Start: 1024, End: 2047, Size: 1024, Completed: false, Registry: "reg2"},
			},
		},
		{
			"空切片",
			[]ChunkInfo{},
			[]ChunkResumeInfo{},
		},
		{
			"nil切片",
			nil,
			[]ChunkResumeInfo{},
		},
		{
			"包含Data字段的ChunkInfo",
			[]ChunkInfo{
				{Index: 0, Start: 0, End: 99, Size: 100, Completed: true, Registry: "reg1", Data: []byte("test")},
			},
			[]ChunkResumeInfo{
				// Data字段不应出现在ResumeInfo中
				{Index: 0, Start: 0, End: 99, Size: 100, Completed: true, Registry: "reg1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertChunksToResumeInfo(tt.chunks)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === ConvertResumeInfoToChunks 测试 ===

func TestConvertResumeInfoToChunks(t *testing.T) {
	tests := []struct {
		name          string
		resumeChunks []ChunkResumeInfo
		want          []ChunkInfo
	}{
		{
			"正常转换",
			[]ChunkResumeInfo{
				{Index: 0, Start: 0, End: 1023, Size: 1024, Completed: true, Registry: "reg1"},
				{Index: 1, Start: 1024, End: 2047, Size: 1024, Completed: false, Registry: "reg2"},
			},
			[]ChunkInfo{
				{Index: 0, Start: 0, End: 1023, Size: 1024, Completed: true, Registry: "reg1", Data: nil},
				{Index: 1, Start: 1024, End: 2047, Size: 1024, Completed: false, Registry: "reg2", Data: nil},
			},
		},
		{
			"空切片",
			[]ChunkResumeInfo{},
			[]ChunkInfo{},
		},
		{
			"nil切片",
			nil,
			[]ChunkInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertResumeInfoToChunks(tt.resumeChunks)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === 双向转换一致性测试 ===

func TestChunkConversionRoundTrip(t *testing.T) {
	// 测试 ChunkInfo -> ChunkResumeInfo -> ChunkInfo 往返转换的一致性
	original := []ChunkInfo{
		{Index: 0, Start: 0, End: 524287, Size: 524288, Completed: true, Registry: "reg1"},
		{Index: 1, Start: 524288, End: 1048575, Size: 524288, Completed: false, Registry: "reg2"},
		{Index: 2, Start: 1048576, End: 1572863, Size: 524288, Completed: false, Registry: ""},
	}

	// 正向转换
	resumeInfo := ConvertChunksToResumeInfo(original)
	assert.Equal(t, len(original), len(resumeInfo))

	// 反向转换
	roundTrip := ConvertResumeInfoToChunks(resumeInfo)

	// 验证核心字段一致（Data字段在往返后会丢失，这是预期行为）
	for i := range original {
		assert.Equal(t, original[i].Index, roundTrip[i].Index, "Index不一致")
		assert.Equal(t, original[i].Start, roundTrip[i].Start, "Start不一致")
		assert.Equal(t, original[i].End, roundTrip[i].End, "End不一致")
		assert.Equal(t, original[i].Size, roundTrip[i].Size, "Size不一致")
		assert.Equal(t, original[i].Completed, roundTrip[i].Completed, "Completed不一致")
		assert.Equal(t, original[i].Registry, roundTrip[i].Registry, "Registry不一致")
	}
}
