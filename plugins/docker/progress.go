package docker

import (
	"fmt"
	"sync"
	"time"
)

// ProgressTracker 进度跟踪器
type ProgressTracker struct {
	mu             sync.RWMutex
	layers         map[string]*LayerProgress
	totalSize      int64
	downloadedSize int64
	startTime      time.Time
	lastUpdate     time.Time
}

// LayerProgress Layer 下载进度
type LayerProgress struct {
	Digest         string
	TotalSize      int64
	DownloadedSize int64
	StartTime      time.Time
	Completed      bool
}

// NewProgressTracker 创建进度跟踪器
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		layers:     make(map[string]*LayerProgress),
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

// StartLayer 开始跟踪 Layer
func (pt *ProgressTracker) StartLayer(digest string, size int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.layers[digest] = &LayerProgress{
		Digest:    digest,
		TotalSize: size,
		StartTime: time.Now(),
	}

	pt.totalSize += size
}

// UpdateProgress 更新进度
func (pt *ProgressTracker) UpdateProgress(digest string, downloaded int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if layer, ok := pt.layers[digest]; ok {
		layer.DownloadedSize += downloaded
		pt.downloadedSize += downloaded
		pt.lastUpdate = time.Now()
	}
}

// CompleteLayer 完成 Layer
func (pt *ProgressTracker) CompleteLayer(digest string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if layer, ok := pt.layers[digest]; ok {
		layer.Completed = true
	}
}

// GetProgress 获取总进度
func (pt *ProgressTracker) GetProgress() (downloaded, total int64, percentage float64) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.totalSize == 0 {
		return 0, 0, 0
	}

	percentage = float64(pt.downloadedSize) / float64(pt.totalSize) * 100
	return pt.downloadedSize, pt.totalSize, percentage
}

// GetSpeed 获取下载速度（字节/秒）
func (pt *ProgressTracker) GetSpeed() float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	duration := time.Since(pt.startTime).Seconds()
	if duration == 0 {
		return 0
	}

	return float64(pt.downloadedSize) / duration
}

// GetETA 获取预计剩余时间
func (pt *ProgressTracker) GetETA() time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	speed := pt.GetSpeed()
	if speed == 0 {
		return 0
	}

	remaining := pt.totalSize - pt.downloadedSize
	seconds := float64(remaining) / speed

	return time.Duration(seconds) * time.Second
}

// GetSummary 获取进度摘要
func (pt *ProgressTracker) GetSummary() string {
	downloaded, total, percentage := pt.GetProgress()
	speed := pt.GetSpeed()
	eta := pt.GetETA()

	return fmt.Sprintf("进度: %.2f%% (%s / %s) | 速度: %.2f MB/s | 剩余: %v",
		percentage,
		formatSize(downloaded),
		formatSize(total),
		speed/1024/1024,
		eta.Round(time.Second),
	)
}

// PrintProgress 打印进度（用于终端输出）
func (pt *ProgressTracker) PrintProgress() {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	// 清除当前行
	fmt.Print("\r\033[K")

	// 打印进度条
	downloaded, total, percentage := pt.GetProgress()
	speed := pt.GetSpeed()

	barWidth := 50
	completed := int(percentage / 100 * float64(barWidth))
	remaining := barWidth - completed

	bar := ""
	for i := 0; i < completed; i++ {
		bar += "="
	}
	if remaining > 0 {
		bar += ">"
		remaining--
	}
	for i := 0; i < remaining; i++ {
		bar += " "
	}

	fmt.Printf("[%s] %.1f%% | %s / %s | %.2f MB/s",
		bar,
		percentage,
		formatSize(downloaded),
		formatSize(total),
		speed/1024/1024,
	)
}

// StartProgressDisplay 启动进度显示（定时刷新）
func (pt *ProgressTracker) StartProgressDisplay(interval time.Duration) chan struct{} {
	stopChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pt.PrintProgress()
			case <-stopChan:
				// 最后打印一次完整进度
				pt.PrintProgress()
				fmt.Println() // 换行
				return
			}
		}
	}()

	return stopChan
}
