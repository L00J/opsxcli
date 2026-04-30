package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// compose.go 纯函数 — 补充
// ---------------------------------------------------------------------------

func TestAppendComposeFile_NilArgs(t *testing.T) {
	result := appendComposeFile(nil, "docker-compose.yml")
	assert.Contains(t, result, "-f")
}

func TestAppendComposeProject_NilArgs(t *testing.T) {
	result := appendComposeProject(nil, "myproject")
	assert.Contains(t, result, "-p")
}

// ---------------------------------------------------------------------------
// monitor.go 纯函数补充
// ---------------------------------------------------------------------------

func TestParseSizeString_More(t *testing.T) {
	assert.Equal(t, int64(1048576), parseSizeString("1MiB"))
	assert.Equal(t, int64(0), parseSizeString(""))
}

// ---------------------------------------------------------------------------
// image.go 补充
// ---------------------------------------------------------------------------

func TestParseImageInspect_MinimalRootFS(t *testing.T) {
	raw := map[string]interface{}{
		"Id":       "sha256:abc",
		"RepoTags": []interface{}{},
		"Created":  "2024-01-01T00:00:00Z",
		"Size":     float64(0),
		"Config":   map[string]interface{}{},
		"RootFS":   map[string]interface{}{},
	}
	result := parseImageInspect(raw)
	assert.NotNil(t, result)
	assert.Equal(t, "sha256:abc", result.ID)
}

func TestParseContainerConfig_Nil(t *testing.T) {
	result := parseContainerConfig(nil)
	assert.Equal(t, ContainerConfigInfo{}, result)
}

func TestParseContainerConfig_Full(t *testing.T) {
	config := map[string]interface{}{
		"Cmd":          []interface{}{"/bin/sh"},
		"ExposedPorts": map[string]interface{}{"80/tcp": map[string]interface{}{}},
		"Env":          []interface{}{"PATH=/usr/bin"},
		"WorkingDir":   "/app",
		"Entrypoint":   []interface{}{"/entrypoint.sh"},
		"User":         "nginx",
		"Labels":       map[string]interface{}{"version": "1.0"},
	}
	result := parseContainerConfig(config)
	assert.Contains(t, result.Cmd, "/bin/sh")
	assert.Equal(t, "/app", result.WorkingDir)
	assert.Equal(t, "nginx", result.User)
}

// ---------------------------------------------------------------------------
// downloader.go 补充 — checkCache/loadResumeInfo/saveResumeInfo
// ---------------------------------------------------------------------------

func TestNewMultiSourceDownloader(t *testing.T) {
	d := NewMultiSourceDownloader([]string{"registry1", "registry2"}, t.TempDir())
	assert.NotNil(t, d)
	assert.Len(t, d.Registries, 2)
}

func TestLoadResumeInfo_Empty(t *testing.T) {
	d := NewMultiSourceDownloader(nil, t.TempDir())
	info := d.loadResumeInfo("sha256:nonexistent")
	assert.Empty(t, info)
}

func TestSaveAndLoadResumeInfo(t *testing.T) {
	d := NewMultiSourceDownloader(nil, t.TempDir())
	chunks := []ChunkInfo{
		{Index: 0, Start: 0, End: 100, Completed: true},
		{Index: 1, Start: 100, End: 200, Completed: false},
	}
	d.saveResumeInfo("sha256:test123", chunks)
	loaded := d.loadResumeInfo("sha256:test123")
	assert.Len(t, loaded, 2)
	assert.True(t, loaded[0].Completed)
	assert.False(t, loaded[1].Completed)
}

func TestDeleteResumeInfo(t *testing.T) {
	d := NewMultiSourceDownloader(nil, t.TempDir())
	d.saveResumeInfo("sha256:todel", []ChunkInfo{{Index: 0, Start: 0, End: 100}})
	d.deleteResumeInfo("sha256:todel")
	info := d.loadResumeInfo("sha256:todel")
	assert.Empty(t, info)
}

func TestCalculateChunks_Basic(t *testing.T) {
	d := NewMultiSourceDownloader(nil, t.TempDir())
	chunks := d.calculateChunks(1000, nil)
	assert.True(t, len(chunks) > 0)
}

func TestCalculateChunks_WithResume(t *testing.T) {
	d := NewMultiSourceDownloader(nil, t.TempDir())
	resume := map[int]*ChunkInfo{0: {Index: 0, Start: 0, End: 500, Completed: true}}
	chunks := d.calculateChunks(1000, resume)
	assert.True(t, len(chunks) > 0)
}
