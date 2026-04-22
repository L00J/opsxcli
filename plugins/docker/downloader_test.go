package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// NewMultiSourceDownloader
// ---------------------------------------------------------------------------

func TestDownloaderNewMultiSourceDownloader(t *testing.T) {
	tmpDir := t.TempDir()

	registries := []string{"https://registry-1.example.com", "https://registry-2.example.com"}
	d := NewMultiSourceDownloader(registries, tmpDir)

	if d == nil {
		t.Fatal("NewMultiSourceDownloader 返回 nil")
	}
	if d.CacheDir != tmpDir {
		t.Errorf("CacheDir = %q, 期望 %q", d.CacheDir, tmpDir)
	}
	if len(d.Registries) != len(registries) {
		t.Errorf("Registries 数量 = %d, 期望 %d", len(d.Registries), len(registries))
	}
	if len(d.Clients) != len(registries) {
		t.Errorf("Clients 数量 = %d, 期望 %d", len(d.Clients), len(registries))
	}
	if d.ChunkSize != 4*1024*1024 {
		t.Errorf("ChunkSize = %d, 期望 %d", d.ChunkSize, 4*1024*1024)
	}
	if d.MaxConcurrency != 5 {
		t.Errorf("MaxConcurrency = %d, 期望 %d", d.MaxConcurrency, 5)
	}
	if d.HealthMonitor == nil {
		t.Error("HealthMonitor 不应为 nil")
	}
	if d.ResumeCache == nil {
		t.Error("ResumeCache 不应为 nil")
	}
}

func TestDownloaderNewMultiSourceDownloaderEmptyRegistries(t *testing.T) {
	tmpDir := t.TempDir()
	d := NewMultiSourceDownloader([]string{}, tmpDir)
	if d == nil {
		t.Fatal("NewMultiSourceDownloader 返回 nil")
	}
	if len(d.Clients) != 0 {
		t.Errorf("空 registries 时 Clients 数量 = %d, 期望 0", len(d.Clients))
	}
}

// ---------------------------------------------------------------------------
// mergeChunks
// ---------------------------------------------------------------------------

func TestDownloaderMergeChunks(t *testing.T) {
	tmpDir := t.TempDir()
	d := &MultiSourceDownloader{CacheDir: tmpDir}

	tests := []struct {
		name        string
		chunks      []ChunkInfo
		expectData  []byte
		expectError bool
	}{
		{
			name:        "空 chunks 列表",
			chunks:      []ChunkInfo{},
			expectData:  []byte{},
			expectError: false,
		},
		{
			name: "单个 chunk",
			chunks: []ChunkInfo{
				{Index: 0, Data: []byte("hello")},
			},
			expectData:  []byte("hello"),
			expectError: false,
		},
		{
			name: "多个 chunks 按顺序合并",
			chunks: []ChunkInfo{
				{Index: 0, Data: []byte("hel")},
				{Index: 1, Data: []byte("lo ")},
				{Index: 2, Data: []byte("world")},
			},
			expectData:  []byte("hello world"),
			expectError: false,
		},
		{
			name: "包含 nil Data 的 chunk",
			chunks: []ChunkInfo{
				{Index: 0, Data: []byte("A")},
				{Index: 1, Data: nil},
				{Index: 2, Data: []byte("B")},
			},
			expectData:  []byte("AB"),
			expectError: false,
		},
		{
			name: "包含空 Data 的 chunk",
			chunks: []ChunkInfo{
				{Index: 0, Data: []byte{}},
				{Index: 1, Data: []byte("data")},
			},
			expectData:  []byte("data"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile := filepath.Join(tmpDir, "merge_output_"+tt.name)
			err := d.mergeChunks(outputFile, tt.chunks)

			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但得到 nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望错误: %v", err)
			}

			data, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("读取输出文件失败: %v", err)
			}
			if string(data) != string(tt.expectData) {
				t.Errorf("合并数据 = %q, 期望 %q", string(data), string(tt.expectData))
			}
		})
	}
}

func TestDownloaderMergeChunksInvalidPath(t *testing.T) {
	d := &MultiSourceDownloader{}
	chunks := []ChunkInfo{
		{Index: 0, Data: []byte("test")},
	}
	// 写入不存在的目录应该失败
	err := d.mergeChunks("/nonexistent/dir/output.dat", chunks)
	if err == nil {
		t.Error("期望写入无效路径时返回错误")
	}
}

// ---------------------------------------------------------------------------
// verifyLayer
// ---------------------------------------------------------------------------

func TestDownloaderVerifyLayer(t *testing.T) {
	tmpDir := t.TempDir()
	d := &MultiSourceDownloader{CacheDir: tmpDir}

	// 创建一个临时文件并计算其 SHA256
	content := []byte("verify layer content test")
	expectedHash := sha256.Sum256(content)
	expectedDigest := "sha256:" + hex.EncodeToString(expectedHash[:])
	wrongDigest := "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	validFile := filepath.Join(tmpDir, "valid_layer")
	if err := os.WriteFile(validFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	tests := []struct {
		name           string
		filePath       string
		expectedDigest string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "digest 匹配",
			filePath:       validFile,
			expectedDigest: expectedDigest,
			expectError:    false,
		},
		{
			name:           "digest 不匹配",
			filePath:       validFile,
			expectedDigest: wrongDigest,
			expectError:    true,
			errorContains:  "校验失败",
		},
		{
			name:           "无效 digest 格式-无冒号",
			filePath:       validFile,
			expectedDigest: "invalidnocolon",
			expectError:    true,
			errorContains:  "无效的 digest 格式",
		},
		{
			name:           "无效 digest 格式-多个冒号",
			filePath:       validFile,
			expectedDigest: "sha256:abc:123",
			expectError:    true,
			errorContains:  "无效的 digest 格式",
		},
		{
			name:           "空 digest",
			filePath:       validFile,
			expectedDigest: "",
			expectError:    true,
			errorContains:  "无效的 digest 格式",
		},
		{
			name:           "文件不存在",
			filePath:       filepath.Join(tmpDir, "nonexistent"),
			expectedDigest: "sha256:abc123",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.verifyLayer(tt.filePath, tt.expectedDigest)
			if tt.expectError {
				if err == nil {
					t.Error("期望返回错误，但得到 nil")
				}
				if tt.errorContains != "" && err != nil {
					if !contains(err.Error(), tt.errorContains) {
						t.Errorf("错误 %q 应包含 %q", err.Error(), tt.errorContains)
					}
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误: %v", err)
				}
			}
		})
	}
}

func TestDownloaderVerifyLayerEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	d := &MultiSourceDownloader{CacheDir: tmpDir}

	// 空文件的 SHA256
	emptyHash := sha256.Sum256([]byte{})
	emptyDigest := "sha256:" + hex.EncodeToString(emptyHash[:])

	emptyFile := filepath.Join(tmpDir, "empty_layer")
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatalf("创建空文件失败: %v", err)
	}

	err := d.verifyLayer(emptyFile, emptyDigest)
	if err != nil {
		t.Errorf("空文件验证应该成功: %v", err)
	}
}

func TestDownloaderVerifyLayerLargeContent(t *testing.T) {
	tmpDir := t.TempDir()
	d := &MultiSourceDownloader{CacheDir: tmpDir}

	// 创建较大的内容
	largeContent := make([]byte, 1024*100) // 100KB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	largeHash := sha256.Sum256(largeContent)
	largeDigest := "sha256:" + hex.EncodeToString(largeHash[:])

	largeFile := filepath.Join(tmpDir, "large_layer")
	if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
		t.Fatalf("创建大文件失败: %v", err)
	}

	err := d.verifyLayer(largeFile, largeDigest)
	if err != nil {
		t.Errorf("大文件验证应该成功: %v", err)
	}
}

// ---------------------------------------------------------------------------
// loadResumeInfo
// ---------------------------------------------------------------------------

func TestDownloaderLoadResumeInfoNilCache(t *testing.T) {
	d := &MultiSourceDownloader{ResumeCache: nil}
	result := d.loadResumeInfo("sha256:abc123")
	if result != nil {
		t.Errorf("ResumeCache 为 nil 时应返回 nil, 得到 %v", result)
	}
}

func TestDownloaderLoadResumeInfoNoSavedData(t *testing.T) {
	tmpDir := t.TempDir()
	rc, err := NewResumeCache(tmpDir)
	if err != nil {
		t.Fatalf("NewResumeCache 失败: %v", err)
	}
	d := &MultiSourceDownloader{ResumeCache: rc}

	result := d.loadResumeInfo("sha256:nonexistent1234567890")
	if result != nil {
		t.Errorf("无已保存数据时应返回 nil, 得到 %v", result)
	}
}

func TestDownloaderLoadResumeInfoWithSavedData(t *testing.T) {
	tmpDir := t.TempDir()
	rc, err := NewResumeCache(tmpDir)
	if err != nil {
		t.Fatalf("NewResumeCache 失败: %v", err)
	}
	d := &MultiSourceDownloader{ResumeCache: rc}

	digest := "sha256:abcdef1234567890"

	// 先保存数据
	chunks := []ChunkInfo{
		{Index: 0, Start: 0, End: 99, Size: 100, Completed: true, Registry: "reg1"},
		{Index: 1, Start: 100, End: 199, Size: 100, Completed: false, Registry: ""},
	}
	d.saveResumeInfo(digest, chunks)

	// 加载数据
	result := d.loadResumeInfo(digest)
	if result == nil {
		t.Fatal("加载已保存数据返回 nil")
	}
	if len(result) != 2 {
		t.Fatalf("加载的 chunk 数量 = %d, 期望 2", len(result))
	}
	if !result[0].Completed {
		t.Error("Chunk 0 应已完成")
	}
	if result[0].Registry != "reg1" {
		t.Errorf("Chunk 0 Registry = %q, 期望 %q", result[0].Registry, "reg1")
	}
	if result[1].Completed {
		t.Error("Chunk 1 不应完成")
	}
	if result[1].Start != 100 {
		t.Errorf("Chunk 1 Start = %d, 期望 100", result[1].Start)
	}
	if result[1].End != 199 {
		t.Errorf("Chunk 1 End = %d, 期望 199", result[1].End)
	}
}

// ---------------------------------------------------------------------------
// saveResumeInfo
// ---------------------------------------------------------------------------

func TestDownloaderSaveResumeInfoNilCache(t *testing.T) {
	d := &MultiSourceDownloader{ResumeCache: nil}
	// 不应 panic
	d.saveResumeInfo("sha256:abc", []ChunkInfo{})
}

func TestDownloaderSaveResumeInfoNormal(t *testing.T) {
	tmpDir := t.TempDir()
	rc, err := NewResumeCache(tmpDir)
	if err != nil {
		t.Fatalf("NewResumeCache 失败: %v", err)
	}
	d := &MultiSourceDownloader{ResumeCache: rc}

	digest := "sha256:savedigest123"
	chunks := []ChunkInfo{
		{Index: 0, Start: 0, End: 499, Size: 500, Completed: true},
		{Index: 1, Start: 500, End: 999, Size: 500, Completed: false},
	}

	d.saveResumeInfo(digest, chunks)

	// 验证数据已写入
	if !rc.Exists(digest) {
		t.Error("保存后 Exists 应返回 true")
	}

	// 加载验证 TotalSize
	info, err := rc.Load(digest)
	if err != nil {
		t.Fatalf("加载保存的数据失败: %v", err)
	}
	if info.TotalSize != 1000 {
		t.Errorf("TotalSize = %d, 期望 1000", info.TotalSize)
	}
	if len(info.Chunks) != 2 {
		t.Errorf("Chunks 数量 = %d, 期望 2", len(info.Chunks))
	}
	if info.Digest != digest {
		t.Errorf("Digest = %q, 期望 %q", info.Digest, digest)
	}
}

func TestDownloaderSaveResumeInfoEmptyChunks(t *testing.T) {
	tmpDir := t.TempDir()
	rc, err := NewResumeCache(tmpDir)
	if err != nil {
		t.Fatalf("NewResumeCache 失败: %v", err)
	}
	d := &MultiSourceDownloader{ResumeCache: rc}

	digest := "sha256:emptychunks"
	d.saveResumeInfo(digest, []ChunkInfo{})

	info, err := rc.Load(digest)
	if err != nil {
		t.Fatalf("加载空 chunks 数据失败: %v", err)
	}
	if info.TotalSize != 0 {
		t.Errorf("TotalSize = %d, 期望 0", info.TotalSize)
	}
	if len(info.Chunks) != 0 {
		t.Errorf("Chunks 数量 = %d, 期望 0", len(info.Chunks))
	}
}

// ---------------------------------------------------------------------------
// deleteResumeInfo
// ---------------------------------------------------------------------------

func TestDownloaderDeleteResumeInfoNilCache(t *testing.T) {
	d := &MultiSourceDownloader{ResumeCache: nil}
	// 不应 panic
	d.deleteResumeInfo("sha256:abc")
}

func TestDownloaderDeleteResumeInfoNormal(t *testing.T) {
	tmpDir := t.TempDir()
	rc, err := NewResumeCache(tmpDir)
	if err != nil {
		t.Fatalf("NewResumeCache 失败: %v", err)
	}
	d := &MultiSourceDownloader{ResumeCache: rc}

	digest := "sha256:todelete123456"

	// 保存后删除
	chunks := []ChunkInfo{
		{Index: 0, Start: 0, End: 99, Size: 100, Completed: true},
	}
	d.saveResumeInfo(digest, chunks)

	if !rc.Exists(digest) {
		t.Fatal("保存后应存在")
	}

	d.deleteResumeInfo(digest)

	if rc.Exists(digest) {
		t.Error("删除后不应存在")
	}
}

func TestDownloaderDeleteResumeInfoNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	rc, err := NewResumeCache(tmpDir)
	if err != nil {
		t.Fatalf("NewResumeCache 失败: %v", err)
	}
	d := &MultiSourceDownloader{ResumeCache: rc}

	// 删除不存在的记录不应 panic 或报错
	d.deleteResumeInfo("sha256:doesnotexist")
}

// ---------------------------------------------------------------------------
// save + load 往返测试
// ---------------------------------------------------------------------------

func TestDownloaderSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	rc, err := NewResumeCache(tmpDir)
	if err != nil {
		t.Fatalf("NewResumeCache 失败: %v", err)
	}
	d := &MultiSourceDownloader{ResumeCache: rc}

	digest := "sha256:roundtrip1234567"
	original := []ChunkInfo{
		{Index: 0, Start: 0, End: 999, Size: 1000, Completed: true, Registry: "https://reg1"},
		{Index: 1, Start: 1000, End: 1999, Size: 1000, Completed: true, Registry: "https://reg2"},
		{Index: 2, Start: 2000, End: 2999, Size: 1000, Completed: false, Registry: ""},
		{Index: 3, Start: 3000, End: 3499, Size: 500, Completed: false, Registry: ""},
	}

	d.saveResumeInfo(digest, original)
	loaded := d.loadResumeInfo(digest)

	if loaded == nil {
		t.Fatal("loadResumeInfo 返回 nil")
	}
	if len(loaded) != len(original) {
		t.Fatalf("加载的 chunk 数量 = %d, 期望 %d", len(loaded), len(original))
	}

	for i, orig := range original {
		got, ok := loaded[i]
		if !ok {
			t.Errorf("缺少 Index %d 的 chunk", i)
			continue
		}
		if got.Index != orig.Index {
			t.Errorf("Chunk %d Index = %d, 期望 %d", i, got.Index, orig.Index)
		}
		if got.Start != orig.Start {
			t.Errorf("Chunk %d Start = %d, 期望 %d", i, got.Start, orig.Start)
		}
		if got.End != orig.End {
			t.Errorf("Chunk %d End = %d, 期望 %d", i, got.End, orig.End)
		}
		if got.Size != orig.Size {
			t.Errorf("Chunk %d Size = %d, 期望 %d", i, got.Size, orig.Size)
		}
		if got.Completed != orig.Completed {
			t.Errorf("Chunk %d Completed = %v, 期望 %v", i, got.Completed, orig.Completed)
		}
		if got.Registry != orig.Registry {
			t.Errorf("Chunk %d Registry = %q, 期望 %q", i, got.Registry, orig.Registry)
		}
	}
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
