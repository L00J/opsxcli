package docker

import (
	"context"
	"sort"
	"sync"
	"time"

	"opsxcli/internal/logger"
)

// RegistrySpeedTest 镜像源测速结果
type RegistrySpeedTest struct {
	Registry string
	Latency  time.Duration
	Success  bool
	Error    error
}

// SpeedTestRegistries 测试多个镜像源的速度
func SpeedTestRegistries(ctx context.Context, registries []string, timeout time.Duration) []RegistrySpeedTest {
	if len(registries) == 0 {
		return nil
	}

	logger.Info("开始测试 %d 个镜像源的连通性和速度...", len(registries))

	// 并发测试所有镜像源
	var wg sync.WaitGroup
	results := make([]RegistrySpeedTest, len(registries))

	testCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for i, registry := range registries {
		wg.Add(1)
		go func(idx int, reg string) {
			defer wg.Done()

			result := RegistrySpeedTest{
				Registry: reg,
			}

			// 创建客户端并测试
			client := NewRegistryClient(reg)
			latency, err := client.Ping(testCtx)

			if err != nil {
				result.Success = false
				result.Error = err
				logger.Debug("[%s] 测速失败: %v", reg, err)
			} else {
				result.Success = true
				result.Latency = latency
				logger.Debug("[%s] 延迟: %v", reg, latency)
			}

			results[idx] = result
		}(i, registry)
	}

	wg.Wait()

	// 按延迟排序（失败的排在最后）
	sort.Slice(results, func(i, j int) bool {
		// 成功的优先
		if results[i].Success && !results[j].Success {
			return true
		}
		if !results[i].Success && results[j].Success {
			return false
		}

		// 都成功时，按延迟排序
		if results[i].Success && results[j].Success {
			return results[i].Latency < results[j].Latency
		}

		// 都失败时，保持原顺序
		return false
	})

	// 输出测速结果
	logger.Info("镜像源测速结果：")
	for i, result := range results {
		if result.Success {
			logger.Info("  %d. %s - %v", i+1, result.Registry, result.Latency)
		} else {
			logger.Debug("  %d. %s - 失败: %v", i+1, result.Registry, result.Error)
		}
	}

	return results
}

// SelectFastestRegistries 选择最快的 N 个镜像源
func SelectFastestRegistries(results []RegistrySpeedTest, count int) []string {
	if count <= 0 {
		count = len(results)
	}

	var selected []string
	for i := 0; i < len(results) && i < count; i++ {
		if results[i].Success {
			selected = append(selected, results[i].Registry)
		}
	}

	return selected
}

// GetOptimalRegistries 智能选择最优镜像源组合
func GetOptimalRegistries(ctx context.Context, registries []string, maxCount int) []string {
	// 如果没有指定镜像源，使用默认列表
	if len(registries) == 0 {
		registries = defaultRegistries
	}

	// 测速
	results := SpeedTestRegistries(ctx, registries, 5*time.Second)

	// 选择最快的几个
	if maxCount <= 0 {
		maxCount = 3 // 默认使用最快的 3 个
	}

	selected := SelectFastestRegistries(results, maxCount)

	if len(selected) == 0 {
		// 如果测速都失败了，返回原列表
		logger.Error("所有镜像源测速失败，使用默认列表")
		return registries
	}

	logger.Success("已选择 %d 个最快的镜像源", len(selected))
	return selected
}

// RegistryHealth 镜像源健康状态
type RegistryHealth struct {
	Registry      string
	Latency       time.Duration
	SuccessCount  int
	FailureCount  int
	LastCheckTime time.Time
}

// HealthMonitor 健康监控器（可用于动态调整镜像源优先级）
type HealthMonitor struct {
	mu      sync.RWMutex
	healths map[string]*RegistryHealth
}

// NewHealthMonitor 创建健康监控器
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		healths: make(map[string]*RegistryHealth),
	}
}

// RecordSuccess 记录成功
func (hm *HealthMonitor) RecordSuccess(registry string, latency time.Duration) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.healths[registry]
	if !exists {
		health = &RegistryHealth{
			Registry: registry,
		}
		hm.healths[registry] = health
	}

	health.SuccessCount++
	health.Latency = latency
	health.LastCheckTime = time.Now()
}

// RecordFailure 记录失败
func (hm *HealthMonitor) RecordFailure(registry string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	health, exists := hm.healths[registry]
	if !exists {
		health = &RegistryHealth{
			Registry: registry,
		}
		hm.healths[registry] = health
	}

	health.FailureCount++
	health.LastCheckTime = time.Now()
}

// GetSortedRegistries 获取按健康度排序的镜像源列表
func (hm *HealthMonitor) GetSortedRegistries() []string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	// 转换为切片
	var healths []*RegistryHealth
	for _, health := range hm.healths {
		healths = append(healths, health)
	}

	// 按成功率和延迟排序
	sort.Slice(healths, func(i, j int) bool {
		// 计算成功率
		totalI := healths[i].SuccessCount + healths[i].FailureCount
		totalJ := healths[j].SuccessCount + healths[j].FailureCount

		if totalI == 0 {
			return false
		}
		if totalJ == 0 {
			return true
		}

		successRateI := float64(healths[i].SuccessCount) / float64(totalI)
		successRateJ := float64(healths[j].SuccessCount) / float64(totalJ)

		// 优先按成功率排序
		if successRateI != successRateJ {
			return successRateI > successRateJ
		}

		// 成功率相同时，按延迟排序
		return healths[i].Latency < healths[j].Latency
	})

	// 提取镜像源列表
	var registries []string
	for _, health := range healths {
		registries = append(registries, health.Registry)
	}

	return registries
}
