package docker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// === shortDigest 测试 ===

func TestShortDigest(t *testing.T) {
	tests := []struct {
		name   string
		digest string
		want   string
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
		name     string
		image    string
		wantName string
		wantTag  string
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
		name         string
		resumeChunks []ChunkResumeInfo
		want         []ChunkInfo
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

// === ProgressTracker 测试 ===

func TestNewProgressTracker(t *testing.T) {
	pt := NewProgressTracker()
	assert.NotNil(t, pt)
	assert.NotNil(t, pt.layers)

	// 新建的 tracker 应该是零进度
	downloaded, total, pct := pt.GetProgress()
	assert.Equal(t, int64(0), downloaded)
	assert.Equal(t, int64(0), total)
	assert.Equal(t, float64(0), pct)
}

func TestProgressTrackerStartLayer(t *testing.T) {
	pt := NewProgressTracker()

	pt.StartLayer("sha256:abc123", 1024)

	downloaded, total, pct := pt.GetProgress()
	assert.Equal(t, int64(0), downloaded)
	assert.Equal(t, int64(1024), total)
	assert.Equal(t, float64(0), pct)
}

func TestProgressTrackerMultipleLayers(t *testing.T) {
	pt := NewProgressTracker()

	pt.StartLayer("layer1", 1024)
	pt.StartLayer("layer2", 2048)
	pt.StartLayer("layer3", 4096)

	_, total, _ := pt.GetProgress()
	assert.Equal(t, int64(1024+2048+4096), total)
}

func TestProgressTrackerUpdateProgress(t *testing.T) {
	pt := NewProgressTracker()

	pt.StartLayer("layer1", 1024)
	pt.UpdateProgress("layer1", 512)

	downloaded, total, pct := pt.GetProgress()
	assert.Equal(t, int64(512), downloaded)
	assert.Equal(t, int64(1024), total)
	assert.InDelta(t, 50.0, pct, 0.01)
}

func TestProgressTrackerUpdateProgressUnknownLayer(t *testing.T) {
	pt := NewProgressTracker()

	pt.StartLayer("layer1", 1024)
	// 更新不存在的 layer，不应影响进度
	pt.UpdateProgress("unknown", 512)

	downloaded, _, _ := pt.GetProgress()
	assert.Equal(t, int64(0), downloaded)
}

func TestProgressTrackerCompleteLayer(t *testing.T) {
	pt := NewProgressTracker()

	pt.StartLayer("layer1", 1024)
	pt.UpdateProgress("layer1", 512)
	pt.CompleteLayer("layer1")

	// 验证 CompleteLayer 不改变下载量
	downloaded, _, _ := pt.GetProgress()
	assert.Equal(t, int64(512), downloaded)
}

func TestProgressTrackerCompleteLayerUnknown(t *testing.T) {
	pt := NewProgressTracker()
	// 不应 panic
	pt.CompleteLayer("nonexistent")
}

func TestProgressTrackerGetProgressEmpty(t *testing.T) {
	pt := NewProgressTracker()

	downloaded, total, pct := pt.GetProgress()
	assert.Equal(t, int64(0), downloaded)
	assert.Equal(t, int64(0), total)
	assert.Equal(t, float64(0), pct)
}

func TestProgressTrackerGetSpeed(t *testing.T) {
	pt := NewProgressTracker()

	// 模拟一些下载
	pt.StartLayer("layer1", 1024*1024)
	pt.UpdateProgress("layer1", 512*1024)

	speed := pt.GetSpeed()
	// 速度应该 > 0（因为已经过了一段时间）
	assert.True(t, speed >= 0)
}

func TestProgressTrackerGetSpeedNoTime(t *testing.T) {
	pt := NewProgressTracker()
	// 刚创建时时间差非常小，但不应 panic
	speed := pt.GetSpeed()
	assert.True(t, speed >= 0)
}

func TestProgressTrackerGetETA(t *testing.T) {
	pt := NewProgressTracker()

	// 没有下载时 ETA 应为 0
	eta := pt.GetETA()
	assert.Equal(t, time.Duration(0), eta)
}

func TestProgressTrackerGetSummary(t *testing.T) {
	pt := NewProgressTracker()
	pt.StartLayer("layer1", 1024*1024)
	pt.UpdateProgress("layer1", 512*1024)

	summary := pt.GetSummary()
	assert.Contains(t, summary, "进度:")
	assert.Contains(t, summary, "速度:")
	assert.Contains(t, summary, "剩余:")
}

func TestProgressTrackerFullProgress(t *testing.T) {
	pt := NewProgressTracker()

	pt.StartLayer("layer1", 1024)
	pt.UpdateProgress("layer1", 1024)
	pt.CompleteLayer("layer1")

	downloaded, total, pct := pt.GetProgress()
	assert.Equal(t, int64(1024), downloaded)
	assert.Equal(t, int64(1024), total)
	assert.InDelta(t, 100.0, pct, 0.01)
}

// === HealthMonitor 测试 ===

func TestNewHealthMonitor(t *testing.T) {
	hm := NewHealthMonitor()
	assert.NotNil(t, hm)
	assert.NotNil(t, hm.healths)
}

func TestHealthMonitorRecordSuccess(t *testing.T) {
	hm := NewHealthMonitor()

	hm.RecordSuccess("reg1", 50*time.Millisecond)
	hm.RecordSuccess("reg1", 30*time.Millisecond)
	hm.RecordSuccess("reg2", 100*time.Millisecond)

	registries := hm.GetSortedRegistries()
	assert.Equal(t, 2, len(registries))
	// reg1 有更高的成功率 (2次成功) 和更低的延迟 (30ms)
	assert.Equal(t, "reg1", registries[0])
}

func TestHealthMonitorRecordFailure(t *testing.T) {
	hm := NewHealthMonitor()

	hm.RecordSuccess("reg1", 50*time.Millisecond)
	hm.RecordFailure("reg2")
	hm.RecordFailure("reg2")

	registries := hm.GetSortedRegistries()
	assert.Equal(t, 2, len(registries))
	// reg1 有100%成功率，排前面
	assert.Equal(t, "reg1", registries[0])
	assert.Equal(t, "reg2", registries[1])
}

func TestHealthMonitorMixedResults(t *testing.T) {
	hm := NewHealthMonitor()

	// reg1: 1成功 1失败 = 50%
	hm.RecordSuccess("reg1", 100*time.Millisecond)
	hm.RecordFailure("reg1")

	// reg2: 2成功 = 100%
	hm.RecordSuccess("reg2", 200*time.Millisecond)
	hm.RecordSuccess("reg2", 200*time.Millisecond)

	registries := hm.GetSortedRegistries()
	// reg2 成功率更高，应排前面
	assert.Equal(t, "reg2", registries[0])
	assert.Equal(t, "reg1", registries[1])
}

func TestHealthMonitorGetSortedRegistriesEmpty(t *testing.T) {
	hm := NewHealthMonitor()
	registries := hm.GetSortedRegistries()
	assert.Equal(t, 0, len(registries))
}

func TestHealthMonitorRecordSuccessNewRegistry(t *testing.T) {
	hm := NewHealthMonitor()
	hm.RecordSuccess("newreg", 50*time.Millisecond)

	registries := hm.GetSortedRegistries()
	assert.Equal(t, []string{"newreg"}, registries)
}

func TestHealthMonitorRecordFailureNewRegistry(t *testing.T) {
	hm := NewHealthMonitor()
	hm.RecordFailure("badreg")

	registries := hm.GetSortedRegistries()
	assert.Equal(t, []string{"badreg"}, registries)
}

func TestHealthMonitorSameSuccessRate(t *testing.T) {
	hm := NewHealthMonitor()

	// 两个都是100%成功率，reg2延迟更低
	hm.RecordSuccess("reg1", 200*time.Millisecond)
	hm.RecordSuccess("reg2", 50*time.Millisecond)

	registries := hm.GetSortedRegistries()
	assert.Equal(t, "reg2", registries[0])
	assert.Equal(t, "reg1", registries[1])
}

// === RegistryClient.normalizeImageName 测试 ===

func TestNormalizeImageName(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		image    string
		want     string
	}{
		{"简单镜像名", "ghcr.io", "nginx", "nginx"},
		{"带用户路径", "ghcr.io", "myuser/myapp", "myuser/myapp"},
		{"带registry前缀", "ghcr.io", "ghcr.io/org/image", "org/image"},
		{"DockerHub官方镜像", "docker.io", "nginx", "library/nginx"},
		{"DockerHub非官方", "docker.io", "myuser/myapp", "myuser/myapp"},
		{"DockerHub别名官方", "registry-1.docker.io", "redis", "library/redis"},
		{"无点的域名前缀不剥除", "myregistry", "myregistry/myapp", "myregistry/myapp"},
		{"长路径", "ghcr.io", "ghcr.io/org/team/image", "org/team/image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RegistryClient{Registry: tt.registry}
			got := c.normalizeImageName(tt.image)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === RegistryClient.buildManifestURL 测试 ===

func TestBuildManifestURL(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		image    string
		tag      string
		want     string
	}{
		{"DockerHub镜像", "docker.io", "nginx", "latest", "https://registry-1.docker.io/v2/library/nginx/manifests/latest"},
		{"DockerHub别名", "registry-1.docker.io", "nginx", "1.25", "https://registry-1.docker.io/v2/library/nginx/manifests/1.25"},
		{"第三方Registry", "ghcr.io", "org/image", "v1", "https://ghcr.io/v2/org/image/manifests/v1"},
		{"带registry前缀的image", "ghcr.io", "ghcr.io/org/image", "v2", "https://ghcr.io/v2/org/image/manifests/v2"},
		{"自定义Registry", "myreg.example.com", "myapp", "stable", "https://myreg.example.com/v2/myapp/manifests/stable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RegistryClient{Registry: tt.registry}
			got := c.buildManifestURL(tt.image, tt.tag)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === RegistryClient.buildBlobURL 测试 ===

func TestBuildBlobURL(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		image    string
		digest   string
		want     string
	}{
		{"DockerHub blob", "docker.io", "nginx", "sha256:abc123", "https://registry-1.docker.io/v2/library/nginx/blobs/sha256:abc123"},
		{"第三方 blob", "ghcr.io", "org/image", "sha256:def456", "https://ghcr.io/v2/org/image/blobs/sha256:def456"},
		{"带registry前缀", "ghcr.io", "ghcr.io/org/image", "sha256:xyz", "https://ghcr.io/v2/org/image/blobs/sha256:xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &RegistryClient{Registry: tt.registry}
			got := c.buildBlobURL(tt.image, tt.digest)
			assert.Equal(t, tt.want, got)
		})
	}
}

// === calculateChunks 测试 ===

func TestCalculateChunks(t *testing.T) {
	d := &MultiSourceDownloader{ChunkSize: 100}

	tests := []struct {
		name        string
		totalSize   int64
		resumeInfo  map[int]*ChunkInfo
		wantChunks  int
		wantFirst   ChunkInfo
		wantLastEnd int64
	}{
		{
			"小文件不分片",
			50,
			nil,
			1,
			ChunkInfo{Index: 0, Start: 0, End: 49, Size: 50},
			49,
		},
		{
			"恰好一个分片",
			100,
			nil,
			1,
			ChunkInfo{Index: 0, Start: 0, End: 99, Size: 100},
			99,
		},
		{
			"两个分片",
			150,
			nil,
			2,
			ChunkInfo{Index: 0, Start: 0, End: 99, Size: 100},
			149,
		},
		{
			"三个分片有余",
			250,
			nil,
			3,
			ChunkInfo{Index: 0, Start: 0, End: 99, Size: 100},
			249,
		},
		{
			"断点续传跳过已完成",
			200,
			map[int]*ChunkInfo{
				0: {Index: 0, Start: 0, End: 99, Size: 100, Completed: true, Registry: "reg1"},
			},
			2,
			ChunkInfo{Index: 0, Start: 0, End: 99, Size: 100, Completed: true, Registry: "reg1"},
			199,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := d.calculateChunks(tt.totalSize, tt.resumeInfo)
			assert.Equal(t, tt.wantChunks, len(chunks))
			assert.Equal(t, tt.wantFirst.Start, chunks[0].Start)
			assert.Equal(t, tt.wantFirst.End, chunks[0].End)
			assert.Equal(t, tt.wantFirst.Size, chunks[0].Size)
			assert.Equal(t, tt.wantLastEnd, chunks[len(chunks)-1].End)

			// 验证最后一个分片不超过总大小
			assert.True(t, chunks[len(chunks)-1].End < tt.totalSize)
		})
	}
}

func TestCalculateChunksResumeFirstChunk(t *testing.T) {
	d := &MultiSourceDownloader{ChunkSize: 100}

	// 小文件 + 有断点续传
	resumeInfo := map[int]*ChunkInfo{
		0: {Index: 0, Start: 0, End: 49, Size: 50, Completed: true, Registry: "reg1"},
	}
	chunks := d.calculateChunks(50, resumeInfo)
	assert.Equal(t, 1, len(chunks))
	assert.True(t, chunks[0].Completed)
	assert.Equal(t, "reg1", chunks[0].Registry)
}

func TestCalculateChunksNoResumeInfo(t *testing.T) {
	d := &MultiSourceDownloader{ChunkSize: 100}

	chunks := d.calculateChunks(350, nil)
	assert.Equal(t, 4, len(chunks))
	assert.Equal(t, int64(0), chunks[0].Start)
	assert.Equal(t, int64(99), chunks[0].End)
	assert.Equal(t, int64(300), chunks[3].Start)
	assert.Equal(t, int64(349), chunks[3].End)
}

func TestCalculateChunksEmptyResumeInfo(t *testing.T) {
	d := &MultiSourceDownloader{ChunkSize: 100}

	// 空 map 的 resumeInfo
	chunks := d.calculateChunks(150, map[int]*ChunkInfo{})
	assert.Equal(t, 2, len(chunks))
	assert.False(t, chunks[0].Completed)
}

// === NewRegistryClient 测试 ===

func TestNewRegistryClient(t *testing.T) {
	c := NewRegistryClient("ghcr.io")
	assert.Equal(t, "ghcr.io", c.Registry)
	assert.NotNil(t, c.HTTPClient)
	assert.Equal(t, 30*time.Second, c.HTTPClient.Timeout)
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
