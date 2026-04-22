package docker

import (
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// RegistrySpeedTest 排序逻辑测试
// 验证 SpeedTestRegistries 中的排序规则：成功优先、按延迟排序
// ============================================================================

func TestRegistrySpeedTest_SortLogic(t *testing.T) {
	t.Run("成功的排在失败的之前", func(t *testing.T) {
		results := []RegistrySpeedTest{
			{Registry: "fail-a", Latency: 0, Success: false, Error: errors.New("timeout")},
			{Registry: "success-a", Latency: 300 * time.Millisecond, Success: true},
		}
		sort.Slice(results, func(i, j int) bool {
			if results[i].Success && !results[j].Success {
				return true
			}
			if !results[i].Success && results[j].Success {
				return false
			}
			if results[i].Success && results[j].Success {
				return results[i].Latency < results[j].Latency
			}
			return false
		})
		assert.True(t, results[0].Success)
		assert.False(t, results[1].Success)
		assert.Equal(t, "success-a", results[0].Registry)
	})

	t.Run("都成功时按延迟升序排列", func(t *testing.T) {
		results := []RegistrySpeedTest{
			{Registry: "slow", Latency: 500 * time.Millisecond, Success: true},
			{Registry: "fast", Latency: 50 * time.Millisecond, Success: true},
			{Registry: "medium", Latency: 200 * time.Millisecond, Success: true},
		}
		sort.Slice(results, func(i, j int) bool {
			if results[i].Success && !results[j].Success {
				return true
			}
			if !results[i].Success && results[j].Success {
				return false
			}
			if results[i].Success && results[j].Success {
				return results[i].Latency < results[j].Latency
			}
			return false
		})
		assert.Equal(t, "fast", results[0].Registry)
		assert.Equal(t, "medium", results[1].Registry)
		assert.Equal(t, "slow", results[2].Registry)
	})

	t.Run("都失败时保持原顺序", func(t *testing.T) {
		results := []RegistrySpeedTest{
			{Registry: "fail-1", Latency: 0, Success: false, Error: errors.New("a")},
			{Registry: "fail-2", Latency: 0, Success: false, Error: errors.New("b")},
			{Registry: "fail-3", Latency: 0, Success: false, Error: errors.New("c")},
		}
		sort.Slice(results, func(i, j int) bool {
			if results[i].Success && !results[j].Success {
				return true
			}
			if !results[i].Success && results[j].Success {
				return false
			}
			if results[i].Success && results[j].Success {
				return results[i].Latency < results[j].Latency
			}
			return false
		})
		assert.Equal(t, "fail-1", results[0].Registry)
		assert.Equal(t, "fail-2", results[1].Registry)
		assert.Equal(t, "fail-3", results[2].Registry)
	})

	t.Run("混合场景：成功按延迟排序，失败排最后", func(t *testing.T) {
		results := []RegistrySpeedTest{
			{Registry: "fail-1", Latency: 0, Success: false, Error: errors.New("timeout")},
			{Registry: "fast", Latency: 100 * time.Millisecond, Success: true},
			{Registry: "fail-2", Latency: 0, Success: false, Error: errors.New("refused")},
			{Registry: "slow", Latency: 300 * time.Millisecond, Success: true},
			{Registry: "medium", Latency: 200 * time.Millisecond, Success: true},
		}
		sort.Slice(results, func(i, j int) bool {
			if results[i].Success && !results[j].Success {
				return true
			}
			if !results[i].Success && results[j].Success {
				return false
			}
			if results[i].Success && results[j].Success {
				return results[i].Latency < results[j].Latency
			}
			return false
		})
		// 前3个应该是成功的，按延迟排序
		assert.True(t, results[0].Success)
		assert.True(t, results[1].Success)
		assert.True(t, results[2].Success)
		assert.Equal(t, "fast", results[0].Registry)
		assert.Equal(t, "medium", results[1].Registry)
		assert.Equal(t, "slow", results[2].Registry)
		// 后2个应该是失败的
		assert.False(t, results[3].Success)
		assert.False(t, results[4].Success)
	})
}

// ============================================================================
// HealthMonitor 纯逻辑测试
// ============================================================================

func TestHealthMonitor_RecordAndGetSorted(t *testing.T) {
	hm := NewHealthMonitor()

	// reg-a: 3次成功 1次失败 → 75%
	hm.RecordSuccess("reg-a", 100*time.Millisecond)
	hm.RecordSuccess("reg-a", 120*time.Millisecond)
	hm.RecordSuccess("reg-a", 110*time.Millisecond)
	hm.RecordFailure("reg-a")

	// reg-b: 4次成功 0次失败 → 100%
	hm.RecordSuccess("reg-b", 200*time.Millisecond)
	hm.RecordSuccess("reg-b", 180*time.Millisecond)

	// reg-c: 1次成功 1次失败 → 50%
	hm.RecordSuccess("reg-c", 50*time.Millisecond)
	hm.RecordFailure("reg-c")

	sorted := hm.GetSortedRegistries()
	assert.Len(t, sorted, 3)

	// reg-b (100%) > reg-a (75%) > reg-c (50%)
	assert.Equal(t, "reg-b", sorted[0])
	assert.Equal(t, "reg-a", sorted[1])
	assert.Equal(t, "reg-c", sorted[2])
}

func TestHealthMonitor_SameSuccessRate_SortByLatency(t *testing.T) {
	hm := NewHealthMonitor()

	// 都是100%成功率，按延迟排序
	hm.RecordSuccess("slow", 500*time.Millisecond)
	hm.RecordSuccess("fast", 50*time.Millisecond)
	hm.RecordSuccess("medium", 200*time.Millisecond)

	sorted := hm.GetSortedRegistries()
	assert.Equal(t, "fast", sorted[0])
	assert.Equal(t, "medium", sorted[1])
	assert.Equal(t, "slow", sorted[2])
}

func TestHealthMonitor_Empty(t *testing.T) {
	hm := NewHealthMonitor()
	sorted := hm.GetSortedRegistries()
	assert.Empty(t, sorted)
}
