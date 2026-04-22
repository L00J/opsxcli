package docker

import (
	"sync"
	"testing"
)

// TestNewProgressTracker_InitialState 验证初始状态
func TestNewProgressTracker_InitialState(t *testing.T) {
	pt := NewProgressTracker()
	if pt == nil {
		t.Fatal("NewProgressTracker() 返回 nil")
	}
	if pt.layers == nil {
		t.Fatal("layers map 未初始化")
	}
	if len(pt.layers) != 0 {
		t.Fatalf("初始 layers 应为空，实际有 %d 个", len(pt.layers))
	}
	// 空 tracker → GetProgress 返回 0, 0, 0
	downloaded, total, pct := pt.GetProgress()
	if downloaded != 0 || total != 0 || pct != 0 {
		t.Fatalf("空 tracker GetProgress() = (%d, %d, %f)，期望 (0, 0, 0)", downloaded, total, pct)
	}
}

// TestProgressTracker_SingleLayer 单层 1000 字节 → 下载 500 → 50%
func TestProgressTracker_SingleLayer(t *testing.T) {
	pt := NewProgressTracker()
	digest := "sha256:abc123"

	pt.StartLayer(digest, 1000)
	downloaded, total, pct := pt.GetProgress()
	if total != 1000 {
		t.Fatalf("StartLayer 后 total = %d，期望 1000", total)
	}
	if downloaded != 0 {
		t.Fatalf("StartLayer 后 downloaded = %d，期望 0", downloaded)
	}

	pt.UpdateProgress(digest, 500)
	downloaded, total, pct = pt.GetProgress()
	if downloaded != 500 {
		t.Fatalf("UpdateProgress(500) 后 downloaded = %d，期望 500", downloaded)
	}
	if total != 1000 {
		t.Fatalf("total = %d，期望 1000", total)
	}
	// percentage = 500 / 1000 * 100 = 50.0
	if pct < 49.9 || pct > 50.1 {
		t.Fatalf("percentage = %f，期望约 50.0", pct)
	}

	// 验证 layer 内部状态
	layer, ok := pt.layers[digest]
	if !ok {
		t.Fatal("layer 不存在")
	}
	if layer.Completed {
		t.Fatal("layer 不应已完成")
	}

	pt.CompleteLayer(digest)
	if !pt.layers[digest].Completed {
		t.Fatal("CompleteLayer 后 layer.Completed 应为 true")
	}

	// CompleteLayer 不改变 downloaded/total
	downloaded, total, pct = pt.GetProgress()
	if downloaded != 500 || total != 1000 {
		t.Fatalf("CompleteLayer 后 (%d, %d)，期望 (500, 1000)", downloaded, total)
	}
}

// TestProgressTracker_MultipleLayers 多层场景
func TestProgressTracker_MultipleLayers(t *testing.T) {
	pt := NewProgressTracker()
	digest1 := "sha256:layer1"
	digest2 := "sha256:layer2"

	// 启动 2 层: 1000 + 2000 = 3000
	pt.StartLayer(digest1, 1000)
	pt.StartLayer(digest2, 2000)

	_, total, _ := pt.GetProgress()
	if total != 3000 {
		t.Fatalf("2 层后 total = %d，期望 3000", total)
	}

	// 更新 layer1: +300; 更新 layer2: +800 → 总计 1100
	pt.UpdateProgress(digest1, 300)
	pt.UpdateProgress(digest2, 800)

	downloaded, total, pct := pt.GetProgress()
	if downloaded != 1100 {
		t.Fatalf("downloaded = %d，期望 1100", downloaded)
	}
	if total != 3000 {
		t.Fatalf("total = %d，期望 3000", total)
	}
	expectedPct := float64(1100) / float64(3000) * 100
	if pct < expectedPct-0.1 || pct > expectedPct+0.1 {
		t.Fatalf("percentage = %f，期望约 %f", pct, expectedPct)
	}

	// 完成 layer1
	pt.CompleteLayer(digest1)
	if !pt.layers[digest1].Completed {
		t.Fatal("layer1 应已完成")
	}
	if pt.layers[digest2].Completed {
		t.Fatal("layer2 不应已完成")
	}

	// CompleteLayer 不影响数值
	downloaded, total, _ = pt.GetProgress()
	if downloaded != 1100 || total != 3000 {
		t.Fatalf("完成 layer1 后 (%d, %d)，期望 (1100, 3000)", downloaded, total)
	}

	// 继续下载 layer2 至完成
	pt.UpdateProgress(digest2, 1200) // 800 + 1200 = 2000 = layer2 total
	pt.CompleteLayer(digest2)

	downloaded, total, pct = pt.GetProgress()
	if downloaded != 2300 { // 300 + 800 + 1200
		t.Fatalf("全部下载后 downloaded = %d，期望 2300", downloaded)
	}
	if total != 3000 {
		t.Fatalf("total = %d，期望 3000", total)
	}
	expectedFinal := float64(2300) / float64(3000) * 100
	if pct < expectedFinal-0.1 || pct > expectedFinal+0.1 {
		t.Fatalf("最终 percentage = %f，期望约 %f", pct, expectedFinal)
	}
}

// TestProgressTracker_UpdateNonexistentDigest 对不存在的 digest 调用 UpdateProgress 不应 panic
func TestProgressTracker_UpdateNonexistentDigest(t *testing.T) {
	pt := NewProgressTracker()
	// 不应 panic
	pt.UpdateProgress("sha256:nonexistent", 999)
	downloaded, total, pct := pt.GetProgress()
	if downloaded != 0 || total != 0 || pct != 0 {
		t.Fatalf("对不存在的 digest 更新后 (%d, %d, %f)，期望 (0, 0, 0)", downloaded, total, pct)
	}
}

// TestProgressTracker_CompleteNonexistentDigest 对不存在的 digest 调用 CompleteLayer 不应 panic
func TestProgressTracker_CompleteNonexistentDigest(t *testing.T) {
	pt := NewProgressTracker()
	// 不应 panic
	pt.CompleteLayer("sha256:nonexistent")
	// 状态不变
	downloaded, total, pct := pt.GetProgress()
	if downloaded != 0 || total != 0 || pct != 0 {
		t.Fatalf("对不存在的 digest 完成后 (%d, %d, %f)，期望 (0, 0, 0)", downloaded, total, pct)
	}
}

// TestProgressTracker_ConcurrentUpdate 并发安全测试
func TestProgressTracker_ConcurrentUpdate(t *testing.T) {
	pt := NewProgressTracker()
	const numLayers = 10
	const incrementsPerLayer = 100
	const incrementValue int64 = 1

	// 启动 10 层，每层 100 字节
	for i := 0; i < numLayers; i++ {
		digest := "sha256:layer" + string(rune('0'+i))
		pt.StartLayer(digest, int64(incrementsPerLayer))
	}

	var wg sync.WaitGroup
	// 每层启动 1 个 goroutine，做 100 次增量更新（每次 +1）
	for i := 0; i < numLayers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			digest := "sha256:layer" + string(rune('0'+idx))
			for j := 0; j < incrementsPerLayer; j++ {
				pt.UpdateProgress(digest, incrementValue)
			}
		}(i)
	}
	wg.Wait()

	downloaded, total, pct := pt.GetProgress()
	expectedDownloaded := int64(numLayers * incrementsPerLayer * int(incrementValue))
	expectedTotal := int64(numLayers * incrementsPerLayer)

	if downloaded != expectedDownloaded {
		t.Fatalf("并发后 downloaded = %d，期望 %d", downloaded, expectedDownloaded)
	}
	if total != expectedTotal {
		t.Fatalf("total = %d，期望 %d", total, expectedTotal)
	}
	if pct < 99.9 || pct > 100.1 {
		t.Fatalf("并发后 percentage = %f，期望 100.0", pct)
	}
}

// TestProgressTracker_UpdateProgressAccumulative 验证 UpdateProgress 是累加的
func TestProgressTracker_UpdateProgressAccumulative(t *testing.T) {
	pt := NewProgressTracker()
	digest := "sha256:accum"

	pt.StartLayer(digest, 1000)
	pt.UpdateProgress(digest, 200)
	pt.UpdateProgress(digest, 300)
	pt.UpdateProgress(digest, 100)

	downloaded, _, _ := pt.GetProgress()
	if downloaded != 600 { // 200 + 300 + 100
		t.Fatalf("累加 downloaded = %d，期望 600", downloaded)
	}
}

// TestProgressTracker_ZeroSizeLayer 零大小 layer
func TestProgressTracker_ZeroSizeLayer(t *testing.T) {
	pt := NewProgressTracker()
	pt.StartLayer("sha256:zero", 0)

	downloaded, total, pct := pt.GetProgress()
	// total = 0 → GetProgress returns 0,0,0
	if downloaded != 0 || total != 0 || pct != 0 {
		t.Fatalf("零大小 layer: (%d, %d, %f)，期望 (0, 0, 0)", downloaded, total, pct)
	}
}
