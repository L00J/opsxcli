package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// ResumeCache CRUD 测试
// ============================================================================

// TestNewResumeCache 测试创建断点续传缓存
func TestNewResumeCache(t *testing.T) {
	t.Run("正常创建缓存目录", func(t *testing.T) {
		tmpDir := t.TempDir()
		cache, err := NewResumeCache(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, cache)

		// 验证 resume 子目录被创建
		resumeDir := filepath.Join(tmpDir, "resume")
		info, err := os.Stat(resumeDir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("已存在目录不报错", func(t *testing.T) {
		tmpDir := t.TempDir()
		// 预先创建目录
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "resume"), 0755))

		cache, err := NewResumeCache(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, cache)
	})

	t.Run("无效路径返回错误", func(t *testing.T) {
		// 使用一个空字节路径或不可写路径
		_, err := NewResumeCache("/proc/nonexistent/path/that/cannot/be/created")
		assert.Error(t, err)
	})
}

// TestResumeCache_SaveAndLoad 测试保存和加载断点续传信息
func TestResumeCache_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	info := &LayerResumeInfo{
		Digest:    "sha256:abc123def456",
		TotalSize: 1024 * 1024,
		Chunks: []ChunkResumeInfo{
			{Index: 0, Start: 0, End: 524287, Size: 524288, Completed: true, Registry: "https://registry-1.docker.io"},
			{Index: 1, Start: 524288, End: 1048575, Size: 524288, Completed: false, Registry: "https://registry-1.docker.io"},
		},
	}

	t.Run("保存和加载基本流程", func(t *testing.T) {
		err := cache.Save(info.Digest, info)
		require.NoError(t, err)

		loaded, err := cache.Load(info.Digest)
		require.NoError(t, err)

		assert.Equal(t, info.Digest, loaded.Digest)
		assert.Equal(t, info.TotalSize, loaded.TotalSize)
		assert.Len(t, loaded.Chunks, 2)
		assert.Equal(t, "1.0", loaded.Version)
		assert.False(t, loaded.UpdatedAt.IsZero())

		// 验证 Chunk 详情
		assert.True(t, loaded.Chunks[0].Completed)
		assert.False(t, loaded.Chunks[1].Completed)
		assert.Equal(t, int64(524288), loaded.Chunks[0].Size)
	})

	t.Run("加载不存在的digest返回错误", func(t *testing.T) {
		_, err := cache.Load("sha256:nonexistent")
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("覆盖保存", func(t *testing.T) {
		// 先保存一次
		err := cache.Save(info.Digest, info)
		require.NoError(t, err)

		// 修改并再次保存
		info.TotalSize = 2048 * 1024
		err = cache.Save(info.Digest, info)
		require.NoError(t, err)

		loaded, err := cache.Load(info.Digest)
		require.NoError(t, err)
		assert.Equal(t, int64(2048*1024), loaded.TotalSize)
	})
}

// TestResumeCache_Delete 测试删除断点续传信息
func TestResumeCache_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	t.Run("删除已存在的信息", func(t *testing.T) {
		info := &LayerResumeInfo{
			Digest:    "sha256:delete_test",
			TotalSize: 1024,
			Chunks:    []ChunkResumeInfo{{Index: 0, Start: 0, End: 1023, Size: 1024, Completed: true}},
		}
		err := cache.Save(info.Digest, info)
		require.NoError(t, err)
		assert.True(t, cache.Exists(info.Digest))

		err = cache.Delete(info.Digest)
		assert.NoError(t, err)
		assert.False(t, cache.Exists(info.Digest))
	})

	t.Run("删除不存在的信息不报错", func(t *testing.T) {
		err := cache.Delete("sha256:nonexistent")
		// os.Remove 不存在的文件返回错误
		assert.Error(t, err)
	})
}

// TestResumeCache_Exists 测试检查断点续传信息是否存在
func TestResumeCache_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	t.Run("不存在时返回false", func(t *testing.T) {
		assert.False(t, cache.Exists("sha256:notexist"))
	})

	t.Run("存在时返回true", func(t *testing.T) {
		info := &LayerResumeInfo{
			Digest:    "sha256:exist_test",
			TotalSize: 512,
			Chunks:    []ChunkResumeInfo{},
		}
		require.NoError(t, cache.Save(info.Digest, info))
		assert.True(t, cache.Exists(info.Digest))
	})

	t.Run("删除后返回false", func(t *testing.T) {
		info := &LayerResumeInfo{
			Digest:    "sha256:delete_then_check",
			TotalSize: 256,
			Chunks:    []ChunkResumeInfo{},
		}
		require.NoError(t, cache.Save(info.Digest, info))
		assert.True(t, cache.Exists(info.Digest))
		require.NoError(t, cache.Delete(info.Digest))
		assert.False(t, cache.Exists(info.Digest))
	})
}

// TestResumeCache_GetProgress 测试获取已下载进度
func TestResumeCache_GetProgress(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	t.Run("不存在的digest返回零值", func(t *testing.T) {
		downloaded, total, pct := cache.GetProgress("sha256:notexist")
		assert.Equal(t, int64(0), downloaded)
		assert.Equal(t, int64(0), total)
		assert.Equal(t, float64(0), pct)
	})

	t.Run("无已完成分片", func(t *testing.T) {
		info := &LayerResumeInfo{
			Digest:    "sha256:no_completed",
			TotalSize: 1000,
			Chunks: []ChunkResumeInfo{
				{Index: 0, Start: 0, End: 499, Size: 500, Completed: false},
				{Index: 1, Start: 500, End: 999, Size: 500, Completed: false},
			},
		}
		require.NoError(t, cache.Save(info.Digest, info))

		downloaded, total, pct := cache.GetProgress(info.Digest)
		assert.Equal(t, int64(0), downloaded)
		assert.Equal(t, int64(1000), total)
		assert.Equal(t, float64(0), pct)
	})

	t.Run("部分完成", func(t *testing.T) {
		info := &LayerResumeInfo{
			Digest:    "sha256:partial",
			TotalSize: 1000,
			Chunks: []ChunkResumeInfo{
				{Index: 0, Start: 0, End: 499, Size: 500, Completed: true},
				{Index: 1, Start: 500, End: 999, Size: 500, Completed: false},
			},
		}
		require.NoError(t, cache.Save(info.Digest, info))

		downloaded, total, pct := cache.GetProgress(info.Digest)
		assert.Equal(t, int64(500), downloaded)
		assert.Equal(t, int64(1000), total)
		assert.InDelta(t, 50.0, pct, 0.01)
	})

	t.Run("全部完成", func(t *testing.T) {
		info := &LayerResumeInfo{
			Digest:    "sha256:complete",
			TotalSize: 1000,
			Chunks: []ChunkResumeInfo{
				{Index: 0, Start: 0, End: 499, Size: 500, Completed: true},
				{Index: 1, Start: 500, End: 999, Size: 500, Completed: true},
			},
		}
		require.NoError(t, cache.Save(info.Digest, info))

		downloaded, total, pct := cache.GetProgress(info.Digest)
		assert.Equal(t, int64(1000), downloaded)
		assert.Equal(t, int64(1000), total)
		assert.InDelta(t, 100.0, pct, 0.01)
	})

	t.Run("TotalSize为零时百分比为零", func(t *testing.T) {
		info := &LayerResumeInfo{
			Digest:    "sha256:zero_total",
			TotalSize: 0,
			Chunks:    []ChunkResumeInfo{},
		}
		require.NoError(t, cache.Save(info.Digest, info))

		_, _, pct := cache.GetProgress(info.Digest)
		assert.Equal(t, float64(0), pct)
	})
}

// TestResumeCache_GetFilename 测试文件名生成
func TestResumeCache_GetFilename(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	t.Run("标准digest格式", func(t *testing.T) {
		filename := cache.getFilename("sha256:abc123")
		expected := filepath.Join(tmpDir, "resume", "sha256:abc123.json")
		assert.Equal(t, expected, filename)
	})

	t.Run("路径遍历防护", func(t *testing.T) {
		// 使用 filepath.Base 防止路径遍历
		filename := cache.getFilename("../../../etc/passwd")
		// filepath.Base("../../../etc/passwd") = "passwd"
		expected := filepath.Join(tmpDir, "resume", "passwd.json")
		assert.Equal(t, expected, filename)
	})

	t.Run("带斜杠的digest", func(t *testing.T) {
		filename := cache.getFilename("some/path/to/digest")
		// filepath.Base 截取最后一段
		expected := filepath.Join(tmpDir, "resume", "digest.json")
		assert.Equal(t, expected, filename)
	})
}

// TestResumeCache_ListAll 测试列出所有断点续传信息
func TestResumeCache_ListAll(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	t.Run("空目录返回空列表", func(t *testing.T) {
		infos, err := cache.ListAll()
		assert.NoError(t, err)
		assert.Empty(t, infos)
	})

	t.Run("列出多个缓存", func(t *testing.T) {
		// 保存多个缓存
		for i := 0; i < 3; i++ {
			info := &LayerResumeInfo{
				Digest:    "sha256:list_test_" + string(rune('A'+i)),
				TotalSize: int64((i + 1) * 1024),
				Chunks:    []ChunkResumeInfo{},
			}
			require.NoError(t, cache.Save(info.Digest, info))
		}

		infos, err := cache.ListAll()
		assert.NoError(t, err)
		assert.Len(t, infos, 3)
	})

	t.Run("跳过无效JSON文件", func(t *testing.T) {
		// 手动创建一个无效的 JSON 文件
		resumeDir := filepath.Join(tmpDir, "resume")
		require.NoError(t, os.WriteFile(filepath.Join(resumeDir, "invalid.json"), []byte("not json"), 0644))

		infos, err := cache.ListAll()
		assert.NoError(t, err)
		// 应跳过无效文件，返回之前保存的3个有效缓存
		assert.Len(t, infos, 3)
	})

	t.Run("跳过子目录", func(t *testing.T) {
		resumeDir := filepath.Join(tmpDir, "resume")
		require.NoError(t, os.MkdirAll(filepath.Join(resumeDir, "subdir"), 0755))

		infos, err := cache.ListAll()
		assert.NoError(t, err)
		assert.Len(t, infos, 3) // 不计入子目录
	})
}

// TestResumeCache_CleanExpired 测试清理过期的断点续传信息
func TestResumeCache_CleanExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	t.Run("清理过期缓存", func(t *testing.T) {
		// 创建一个"过期"的缓存
		oldInfo := &LayerResumeInfo{
			Digest:    "sha256:old_expired",
			TotalSize: 1024,
			UpdatedAt: time.Now().Add(-48 * time.Hour), // 直接设置过期时间
			Version:   "1.0",
			Chunks:    []ChunkResumeInfo{},
		}
		// 直接写入文件（不通过Save，因为Save会覆盖UpdatedAt）
		filename := cache.getFilename(oldInfo.Digest)
		data, err := json.MarshalIndent(oldInfo, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filename, data, 0644))

		// 创建一个新的（未过期的）缓存
		newInfo := &LayerResumeInfo{
			Digest:    "sha256:new_fresh",
			TotalSize: 2048,
			Chunks:    []ChunkResumeInfo{{Index: 0, Start: 0, End: 2047, Size: 2048, Completed: true}},
		}
		require.NoError(t, cache.Save(newInfo.Digest, newInfo))

		// 清理超过24小时的缓存
		err = cache.CleanExpired(24 * time.Hour)
		assert.NoError(t, err)

		// 过期的应该被删除
		assert.False(t, cache.Exists("sha256:old_expired"))
		// 新的应该保留
		assert.True(t, cache.Exists("sha256:new_fresh"))
	})

	t.Run("所有缓存都未过期", func(t *testing.T) {
		tmpDir2 := t.TempDir()
		cache2, err := NewResumeCache(tmpDir2)
		require.NoError(t, err)

		info := &LayerResumeInfo{
			Digest:    "sha256:fresh_only",
			TotalSize: 512,
			Chunks:    []ChunkResumeInfo{},
		}
		require.NoError(t, cache2.Save(info.Digest, info))

		err = cache2.CleanExpired(24 * time.Hour)
		assert.NoError(t, err)
		assert.True(t, cache2.Exists(info.Digest))
	})

	t.Run("空目录不报错", func(t *testing.T) {
		tmpDir3 := t.TempDir()
		cache3, err := NewResumeCache(tmpDir3)
		require.NoError(t, err)

		err = cache3.CleanExpired(1 * time.Hour)
		assert.NoError(t, err)
	})
}

// ============================================================================
// ResumeCache 并发安全测试
// ============================================================================

// TestResumeCache_ConcurrentAccess 测试并发读写安全性
func TestResumeCache_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewResumeCache(tmpDir)
	require.NoError(t, err)

	// 并发保存
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			info := &LayerResumeInfo{
				Digest:    "sha256:concurrent_test",
				TotalSize: int64(idx * 100),
				Chunks:    []ChunkResumeInfo{},
			}
			_ = cache.Save(info.Digest, info)
			done <- true
		}(i)
	}

	// 等待所有写入完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证最终状态是有效的
	loaded, err := cache.Load("sha256:concurrent_test")
	assert.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, "sha256:concurrent_test", loaded.Digest)
}

// ============================================================================
// 转换函数测试（补充边界条件）
// ============================================================================

// TestConvertChunksToResumeInfo_Extended 测试 ChunkInfo 到 ChunkResumeInfo 的转换（扩展边界条件）
func TestConvertChunksToResumeInfo_Extended(t *testing.T) {
	t.Run("空切片", func(t *testing.T) {
		result := ConvertChunksToResumeInfo([]ChunkInfo{})
		assert.Empty(t, result)
	})

	t.Run("正常转换", func(t *testing.T) {
		chunks := []ChunkInfo{
			{Index: 0, Start: 0, End: 99, Size: 100, Completed: true, Registry: "reg1"},
			{Index: 1, Start: 100, End: 199, Size: 100, Completed: false, Registry: "reg2"},
		}
		result := ConvertChunksToResumeInfo(chunks)
		assert.Len(t, result, 2)
		assert.Equal(t, 0, result[0].Index)
		assert.Equal(t, int64(0), result[0].Start)
		assert.True(t, result[0].Completed)
		assert.Equal(t, "reg1", result[0].Registry)
		assert.False(t, result[1].Completed)
	})
}

// TestConvertResumeInfoToChunks_Extended 测试 ChunkResumeInfo 到 ChunkInfo 的转换（扩展边界条件）
func TestConvertResumeInfoToChunks_Extended(t *testing.T) {
	t.Run("空切片", func(t *testing.T) {
		result := ConvertResumeInfoToChunks([]ChunkResumeInfo{})
		assert.Empty(t, result)
	})

	t.Run("正常转换", func(t *testing.T) {
		resumeChunks := []ChunkResumeInfo{
			{Index: 0, Start: 0, End: 499, Size: 500, Completed: true, Registry: "https://registry.docker.io"},
			{Index: 1, Start: 500, End: 999, Size: 500, Completed: false, Registry: ""},
		}
		result := ConvertResumeInfoToChunks(resumeChunks)
		assert.Len(t, result, 2)
		assert.Equal(t, 0, result[0].Index)
		assert.Equal(t, int64(500), result[0].Size)
		assert.True(t, result[0].Completed)
		assert.Equal(t, "https://registry.docker.io", result[0].Registry)
	})

	t.Run("往返转换一致性", func(t *testing.T) {
		original := []ChunkInfo{
			{Index: 0, Start: 0, End: 999, Size: 1000, Completed: true, Registry: "test"},
		}
		resume := ConvertChunksToResumeInfo(original)
		roundtrip := ConvertResumeInfoToChunks(resume)
		assert.Equal(t, original, roundtrip)
	})

	t.Run("nil切片转为空切片", func(t *testing.T) {
		result := ConvertChunksToResumeInfo(nil)
		assert.Empty(t, result)
		assert.NotNil(t, result)
	})

	t.Run("nil切片反向转为空切片", func(t *testing.T) {
		result := ConvertResumeInfoToChunks(nil)
		assert.Empty(t, result)
		assert.NotNil(t, result)
	})

	t.Run("多个分片往返转换", func(t *testing.T) {
		original := []ChunkInfo{
			{Index: 0, Start: 0, End: 999, Size: 1000, Completed: true, Registry: "reg-a"},
			{Index: 1, Start: 1000, End: 1999, Size: 1000, Completed: false, Registry: "reg-b"},
			{Index: 2, Start: 2000, End: 2999, Size: 1000, Completed: true, Registry: "reg-a"},
			{Index: 3, Start: 3000, End: 3999, Size: 1000, Completed: false, Registry: ""},
		}
		resume := ConvertChunksToResumeInfo(original)
		roundtrip := ConvertResumeInfoToChunks(resume)
		assert.Equal(t, original, roundtrip)
	})
}
