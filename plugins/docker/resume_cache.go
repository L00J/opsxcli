package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ResumeCache 断点续传缓存
type ResumeCache struct {
	mu       sync.RWMutex
	cacheDir string
}

// LayerResumeInfo Layer 断点续传信息
type LayerResumeInfo struct {
	Digest    string            `json:"digest"`
	TotalSize int64             `json:"total_size"`
	Chunks    []ChunkResumeInfo `json:"chunks"`
	UpdatedAt time.Time         `json:"updated_at"`
	Version   string            `json:"version"` // 缓存格式版本
}

// ChunkResumeInfo 分片断点续传信息
type ChunkResumeInfo struct {
	Index     int    `json:"index"`
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	Size      int64  `json:"size"`
	Completed bool   `json:"completed"`
	Registry  string `json:"registry,omitempty"` // 记录从哪个源下载的
}

// NewResumeCache 创建断点续传缓存
func NewResumeCache(cacheDir string) (*ResumeCache, error) {
	// 确保缓存目录存在
	resumeDir := filepath.Join(cacheDir, "resume")
	if err := os.MkdirAll(resumeDir, 0755); err != nil {
		return nil, err
	}

	return &ResumeCache{
		cacheDir: resumeDir,
	}, nil
}

// Save 保存断点续传信息
func (rc *ResumeCache) Save(digest string, info *LayerResumeInfo) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	info.UpdatedAt = time.Now()
	info.Version = "1.0"

	filename := rc.getFilename(digest)

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// Load 加载断点续传信息
func (rc *ResumeCache) Load(digest string) (*LayerResumeInfo, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	filename := rc.getFilename(digest)

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var info LayerResumeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// Delete 删除断点续传信息
func (rc *ResumeCache) Delete(digest string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	filename := rc.getFilename(digest)
	return os.Remove(filename)
}

// Exists 检查是否存在断点续传信息
func (rc *ResumeCache) Exists(digest string) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	filename := rc.getFilename(digest)
	_, err := os.Stat(filename)
	return err == nil
}

// GetProgress 获取已下载进度
func (rc *ResumeCache) GetProgress(digest string) (downloaded, total int64, percentage float64) {
	info, err := rc.Load(digest)
	if err != nil {
		return 0, 0, 0
	}

	total = info.TotalSize
	for _, chunk := range info.Chunks {
		if chunk.Completed {
			downloaded += chunk.Size
		}
	}

	if total > 0 {
		percentage = float64(downloaded) / float64(total) * 100
	}

	return
}

// getFilename 获取缓存文件名
func (rc *ResumeCache) getFilename(digest string) string {
	// 使用 digest 作为文件名（替换特殊字符）
	filename := digest
	filename = filepath.Base(filename) // 防止路径遍历
	return filepath.Join(rc.cacheDir, filename+".json")
}

// ListAll 列出所有断点续传信息
func (rc *ResumeCache) ListAll() ([]*LayerResumeInfo, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	files, err := os.ReadDir(rc.cacheDir)
	if err != nil {
		return nil, err
	}

	var infos []*LayerResumeInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(rc.cacheDir, file.Name()))
		if err != nil {
			continue
		}

		var info LayerResumeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}

		infos = append(infos, &info)
	}

	return infos, nil
}

// listAllLocked 列出所有断点续传信息（调用者必须持有锁）
func (rc *ResumeCache) listAllLocked() ([]*LayerResumeInfo, error) {
	files, err := os.ReadDir(rc.cacheDir)
	if err != nil {
		return nil, err
	}

	var infos []*LayerResumeInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(rc.cacheDir, file.Name()))
		if err != nil {
			continue
		}

		var info LayerResumeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}

		infos = append(infos, &info)
	}

	return infos, nil
}

// deleteLocked 删除断点续传信息（调用者必须持有锁）
func (rc *ResumeCache) deleteLocked(digest string) error {
	filename := rc.getFilename(digest)
	return os.Remove(filename)
}

// CleanExpired 清理过期的断点续传信息
func (rc *ResumeCache) CleanExpired(maxAge time.Duration) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	infos, err := rc.listAllLocked()
	if err != nil {
		return err
	}

	now := time.Now()
	for _, info := range infos {
		if now.Sub(info.UpdatedAt) > maxAge {
			rc.deleteLocked(info.Digest)
		}
	}

	return nil
}

// ConvertChunksToResumeInfo 将 ChunkInfo 转换为 ChunkResumeInfo
func ConvertChunksToResumeInfo(chunks []ChunkInfo) []ChunkResumeInfo {
	resumeChunks := make([]ChunkResumeInfo, len(chunks))
	for i, chunk := range chunks {
		resumeChunks[i] = ChunkResumeInfo{
			Index:     chunk.Index,
			Start:     chunk.Start,
			End:       chunk.End,
			Size:      chunk.Size,
			Completed: chunk.Completed,
			Registry:  chunk.Registry,
		}
	}
	return resumeChunks
}

// ConvertResumeInfoToChunks 将 ChunkResumeInfo 转换为 ChunkInfo
func ConvertResumeInfoToChunks(resumeChunks []ChunkResumeInfo) []ChunkInfo {
	chunks := make([]ChunkInfo, len(resumeChunks))
	for i, resumeChunk := range resumeChunks {
		chunks[i] = ChunkInfo{
			Index:     resumeChunk.Index,
			Start:     resumeChunk.Start,
			End:       resumeChunk.End,
			Size:      resumeChunk.Size,
			Completed: resumeChunk.Completed,
			Registry:  resumeChunk.Registry,
		}
	}
	return chunks
}
