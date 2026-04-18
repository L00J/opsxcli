package docker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"opsxcli/internal/logger"
)

// MultiSourceDownloader 多源并行下载器
type MultiSourceDownloader struct {
	Registries     []string          // 镜像源列表
	Clients        []*RegistryClient // Registry 客户端列表
	ChunkSize      int64             // 分片大小（默认 4MB）
	MaxConcurrency int               // 最大并发数
	CacheDir       string            // 缓存目录
	HealthMonitor  *HealthMonitor    // 健康监控
	ResumeCache    *ResumeCache      // 断点续传缓存
}

// DownloadResult 下载结果
type DownloadResult struct {
	LayerDigest string
	FilePath    string
	Size        int64
	Duration    time.Duration
	Error       error
}

// ChunkInfo 分片信息
type ChunkInfo struct {
	Index     int
	Start     int64
	End       int64
	Size      int64
	Registry  string
	Completed bool
	Data      []byte
}

// NewMultiSourceDownloader 创建多源下载器
func NewMultiSourceDownloader(registries []string, cacheDir string) *MultiSourceDownloader {
	clients := make([]*RegistryClient, len(registries))
	for i, registry := range registries {
		clients[i] = NewRegistryClient(registry)
	}

	resumeCache, _ := NewResumeCache(cacheDir)

	return &MultiSourceDownloader{
		Registries:     registries,
		Clients:        clients,
		ChunkSize:      4 * 1024 * 1024, // 4MB
		MaxConcurrency: 5,
		CacheDir:       cacheDir,
		HealthMonitor:  NewHealthMonitor(),
		ResumeCache:    resumeCache,
	}
}

// DownloadLayer 下载单个 Layer（支持多源并行和断点续传）
func (d *MultiSourceDownloader) DownloadLayer(ctx context.Context, image string, layer LayerDescriptor, progress *ProgressTracker) (*DownloadResult, error) {
	startTime := time.Now()

	logger.Info("[Layer %s] 开始下载 (大小: %s)", shortDigest(layer.Digest), formatSize(layer.Size))

	// 检查缓存
	cachedPath, err := d.checkCache(layer.Digest, layer.Size)
	if err == nil && cachedPath != "" {
		logger.Info("[Layer %s] 使用缓存", shortDigest(layer.Digest))
		return &DownloadResult{
			LayerDigest: layer.Digest,
			FilePath:    cachedPath,
			Size:        layer.Size,
			Duration:    time.Since(startTime),
		}, nil
	}

	// 创建临时文件
	tempFile := filepath.Join(d.CacheDir, layer.Digest+".tmp")
	finalFile := filepath.Join(d.CacheDir, layer.Digest)

	// 加载断点续传信息
	resumeInfo := d.loadResumeInfo(layer.Digest)

	// 计算分片
	chunks := d.calculateChunks(layer.Size, resumeInfo)

	logger.Info("[Layer %s] 分为 %d 个分片，使用 %d 个镜像源并行下载",
		shortDigest(layer.Digest), len(chunks), len(d.Registries))

	// 初始化进度
	if progress != nil {
		progress.StartLayer(layer.Digest, layer.Size)
	}

	// 并发下载所有分片
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, d.MaxConcurrency)
	errChan := make(chan error, len(chunks))

	for i := range chunks {
		wg.Add(1)
		go func(chunk *ChunkInfo) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 下载分片
			if err := d.downloadChunk(ctx, image, layer.Digest, chunk, progress); err != nil {
				errChan <- fmt.Errorf("分片 %d 下载失败: %v", chunk.Index, err)
				return
			}
		}(&chunks[i])
	}

	wg.Wait()
	close(errChan)

	// 检查错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		// 保存断点续传信息
		d.saveResumeInfo(layer.Digest, chunks)
		return nil, fmt.Errorf("下载失败: %v", errors[0])
	}

	// 合并分片
	if err := d.mergeChunks(tempFile, chunks); err != nil {
		return nil, fmt.Errorf("合并分片失败: %v", err)
	}

	// 验证完整性
	if err := d.verifyLayer(tempFile, layer.Digest); err != nil {
		return nil, fmt.Errorf("验证失败: %v", err)
	}

	// 重命名为最终文件
	if err := os.Rename(tempFile, finalFile); err != nil {
		return nil, fmt.Errorf("重命名失败: %v", err)
	}

	// 删除断点续传信息
	d.deleteResumeInfo(layer.Digest)

	duration := time.Since(startTime)
	speed := float64(layer.Size) / duration.Seconds() / 1024 / 1024 // MB/s

	logger.Success("[Layer %s] 下载完成 (耗时: %v, 速度: %.2f MB/s)",
		shortDigest(layer.Digest), duration, speed)

	if progress != nil {
		progress.CompleteLayer(layer.Digest)
	}

	return &DownloadResult{
		LayerDigest: layer.Digest,
		FilePath:    finalFile,
		Size:        layer.Size,
		Duration:    duration,
	}, nil
}

// downloadChunk 下载单个分片（支持多源重试）
func (d *MultiSourceDownloader) downloadChunk(ctx context.Context, image, digest string, chunk *ChunkInfo, progress *ProgressTracker) error {
	// 如果已经下载过，跳过
	if chunk.Completed {
		return nil
	}

	var lastErr error

	// 尝试所有镜像源
	for _, client := range d.Clients {
		startTime := time.Now()

		data, err := client.DownloadLayer(ctx, image, digest, chunk.Start, chunk.End)
		if err != nil {
			lastErr = err
			d.HealthMonitor.RecordFailure(client.Registry)
			logger.Debug("[Chunk %d] 从 %s 下载失败: %v", chunk.Index, client.Registry, err)
			continue
		}

		// 记录成功
		latency := time.Since(startTime)
		d.HealthMonitor.RecordSuccess(client.Registry, latency)

		// 保存数据
		chunk.Data = data
		chunk.Completed = true
		chunk.Registry = client.Registry

		// 更新进度
		if progress != nil {
			progress.UpdateProgress(digest, int64(len(data)))
		}

		logger.Debug("[Chunk %d] 从 %s 下载成功 (%s)",
			chunk.Index, client.Registry, formatSize(int64(len(data))))

		return nil
	}

	return fmt.Errorf("所有镜像源都失败: %v", lastErr)
}

// calculateChunks 计算分片
func (d *MultiSourceDownloader) calculateChunks(totalSize int64, resumeInfo map[int]*ChunkInfo) []ChunkInfo {
	// 小文件不分片
	if totalSize < d.ChunkSize {
		if resumeInfo != nil && len(resumeInfo) > 0 {
			if chunk, ok := resumeInfo[0]; ok {
				return []ChunkInfo{*chunk}
			}
		}
		return []ChunkInfo{
			{Index: 0, Start: 0, End: totalSize - 1, Size: totalSize},
		}
	}

	// 计算分片数量
	numChunks := int(totalSize / d.ChunkSize)
	if totalSize%d.ChunkSize != 0 {
		numChunks++
	}

	chunks := make([]ChunkInfo, numChunks)

	for i := 0; i < numChunks; i++ {
		start := int64(i) * d.ChunkSize
		end := start + d.ChunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}

		// 检查是否有断点续传信息
		if resumeInfo != nil {
			if resumedChunk, ok := resumeInfo[i]; ok && resumedChunk.Completed {
				chunks[i] = *resumedChunk
				continue
			}
		}

		chunks[i] = ChunkInfo{
			Index: i,
			Start: start,
			End:   end,
			Size:  end - start + 1,
		}
	}

	return chunks
}

// mergeChunks 合并分片
func (d *MultiSourceDownloader) mergeChunks(outputFile string, chunks []ChunkInfo) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, chunk := range chunks {
		if _, err := file.Write(chunk.Data); err != nil {
			return err
		}
	}

	return nil
}

// verifyLayer 验证 Layer 完整性
func (d *MultiSourceDownloader) verifyLayer(filePath, expectedDigest string) error {
	// 提取 digest 值（去掉 sha256: 前缀）
	parts := strings.Split(expectedDigest, ":")
	if len(parts) != 2 {
		return fmt.Errorf("无效的 digest 格式: %s", expectedDigest)
	}

	expectedHash := parts[1]

	// 计算文件 SHA256
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := file.WriteTo(hasher); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("校验失败: 期望 %s, 实际 %s", expectedHash, actualHash)
	}

	return nil
}

// checkCache 检查缓存
func (d *MultiSourceDownloader) checkCache(digest string, expectedSize int64) (string, error) {
	cacheFile := filepath.Join(d.CacheDir, digest)

	info, err := os.Stat(cacheFile)
	if err != nil {
		return "", err
	}

	if info.Size() != expectedSize {
		return "", fmt.Errorf("缓存文件大小不匹配")
	}

	// 验证完整性
	if err := d.verifyLayer(cacheFile, digest); err != nil {
		return "", err
	}

	return cacheFile, nil
}

// loadResumeInfo 加载断点续传信息
func (d *MultiSourceDownloader) loadResumeInfo(digest string) map[int]*ChunkInfo {
	if d.ResumeCache == nil {
		return nil
	}

	info, err := d.ResumeCache.Load(digest)
	if err != nil {
		return nil
	}

	// 转换为 map
	resumeMap := make(map[int]*ChunkInfo)
	for _, chunk := range info.Chunks {
		chunkInfo := ChunkInfo{
			Index:     chunk.Index,
			Start:     chunk.Start,
			End:       chunk.End,
			Size:      chunk.Size,
			Completed: chunk.Completed,
			Registry:  chunk.Registry,
		}
		resumeMap[chunk.Index] = &chunkInfo
	}

	return resumeMap
}

// saveResumeInfo 保存断点续传信息
func (d *MultiSourceDownloader) saveResumeInfo(digest string, chunks []ChunkInfo) {
	if d.ResumeCache == nil {
		return
	}

	info := &LayerResumeInfo{
		Digest:    digest,
		TotalSize: 0,
		Chunks:    ConvertChunksToResumeInfo(chunks),
	}

	// 计算总大小
	for _, chunk := range chunks {
		info.TotalSize += chunk.Size
	}

	d.ResumeCache.Save(digest, info)
}

// deleteResumeInfo 删除断点续传信息
func (d *MultiSourceDownloader) deleteResumeInfo(digest string) {
	if d.ResumeCache == nil {
		return
	}

	d.ResumeCache.Delete(digest)
}

// shortDigest 缩短 digest 显示
func shortDigest(digest string) string {
	parts := strings.Split(digest, ":")
	if len(parts) == 2 {
		return parts[1][:12]
	}
	return digest[:12]
}

// formatSize 格式化大小
func formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
