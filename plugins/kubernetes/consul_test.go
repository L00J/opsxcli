package kubernetes

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== consul.go getServiceHash 测试 ==========

func TestGetServiceHash(t *testing.T) {
	// 创建临时目录用于 KubernetesMonitor
	tmpDir := t.TempDir()
	m := &KubernetesMonitor{
		cacheDir: tmpDir,
	}

	// 相同的namespace/name应该产生相同的hash
	hash1 := m.getServiceHash("default", "my-service")
	hash2 := m.getServiceHash("default", "my-service")
	assert.Equal(t, hash1, hash2, "相同的输入应该产生相同的hash")

	// 不同的namespace/name应该产生不同的hash
	hash3 := m.getServiceHash("kube-system", "my-service")
	assert.NotEqual(t, hash1, hash3, "不同namespace应产生不同hash")

	hash4 := m.getServiceHash("default", "other-service")
	assert.NotEqual(t, hash1, hash4, "不同name应产生不同hash")

	// 验证hash格式（md5 = 32 hex字符）
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte("default/my-service")))
	assert.Equal(t, expectedHash, hash1, "hash应该是namespace/name的md5值")
}

// ========== consul.go 缓存函数测试 ==========

func TestSaveAndLoadCache(t *testing.T) {
	tmpDir := t.TempDir()
	m := &KubernetesMonitor{
		cacheDir: tmpDir,
	}

	// 保存缓存
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": float64(42),
	}
	err := m.saveCache("test-key", testData, 300)
	require.NoError(t, err)

	// 验证缓存文件存在
	cacheFile := filepath.Join(tmpDir, "test-key.json")
	_, err = os.Stat(cacheFile)
	assert.NoError(t, err, "缓存文件应该存在")

	// 加载缓存 - saveCache会将数据包装在{"expire":..., "data":...}中
	loaded, err := m.loadCache("test-key")
	require.NoError(t, err)
	assert.NotNil(t, loaded["expire"], "应该有expire字段")
	assert.NotNil(t, loaded["data"], "应该有data字段")
}

func TestLoadCacheNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	m := &KubernetesMonitor{
		cacheDir: tmpDir,
	}

	// 加载不存在的缓存
	_, err := m.loadCache("nonexistent")
	assert.Error(t, err, "不存在的缓存应该返回错误")
}

func TestIsCacheValid(t *testing.T) {
	tmpDir := t.TempDir()
	m := &KubernetesMonitor{
		cacheDir: tmpDir,
	}

	// 保存有效缓存（TTL 300秒）
	testData := map[string]interface{}{"test": "data"}
	err := m.saveCache("valid-key", testData, 300)
	require.NoError(t, err)

	// 应该有效
	assert.True(t, m.isCacheValid("valid-key"), "未过期的缓存应该有效")

	// 保存已过期的缓存（TTL 0秒，使用过去的时间）
	// saveCache 使用 time.Now().Add(ttl), ttl=0 意味着立即过期
	err = m.saveCache("expired-key", testData, 0)
	require.NoError(t, err)

	// 等待一小段时间确保时间过去
	time.Sleep(10 * time.Millisecond)

	// 应该无效（TTL=0意味着已经过期）
	assert.False(t, m.isCacheValid("expired-key"), "已过期的缓存应该无效")

	// 不存在的缓存应该无效
	assert.False(t, m.isCacheValid("nonexistent"), "不存在的缓存应该无效")
}

func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	m := &KubernetesMonitor{
		cacheDir: tmpDir,
	}

	// 创建一些缓存文件
	testData := map[string]interface{}{"test": "data"}
	err := m.saveCache("cache1", testData, 300)
	require.NoError(t, err)
	err = m.saveCache("cache2", testData, 300)
	require.NoError(t, err)

	// 清除缓存
	m.ClearCache()

	// 目录应该被重建（空的）
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "清除后目录应该为空")
}

// ========== consul.go 结构体测试 ==========

func TestServiceRegistrationStruct(t *testing.T) {
	sr := ServiceRegistration{
		ID:      "my-service-default",
		Name:    "my-service",
		Address: "10.0.0.1",
		Port:    8080,
		Checks: []ServiceCheck{
			{HTTP: "http://10.0.0.1:8080/health", Interval: "10s"},
		},
		Meta: map[string]string{"version": "1.0"},
	}

	assert.Equal(t, "my-service-default", sr.ID)
	assert.Equal(t, 8080, sr.Port)
	assert.Len(t, sr.Checks, 1)
	assert.Equal(t, "1.0", sr.Meta["version"])
}

func TestPodIPCacheStruct(t *testing.T) {
	cache := PodIPCache{
		PodIPs: map[string]string{
			"web-0": "10.0.0.1",
			"web-1": "10.0.0.2",
		},
	}

	assert.Equal(t, "10.0.0.1", cache.PodIPs["web-0"])
	assert.Equal(t, "10.0.0.2", cache.PodIPs["web-1"])
	assert.Len(t, cache.PodIPs, 2)
}

func TestHealthCheckResultStruct(t *testing.T) {
	hcr := HealthCheckResult{
		ServiceID: "svc-123",
	}
	assert.Equal(t, "svc-123", hcr.ServiceID)
}
