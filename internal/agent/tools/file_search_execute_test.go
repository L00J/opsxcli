package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// FileSearchTool.Execute integration tests
// ---------------------------------------------------------------------------

func TestFileSearchExecute_ContentSearchBasic(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("hello world\nfoo bar\nhello again\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("no match here\n"), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "hello",
		"path":    tmpDir,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileSearchResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, "hello", parsed.Pattern)
	assert.Equal(t, "content", parsed.Target)
	assert.Equal(t, 2, parsed.Total)
	assert.Len(t, parsed.Matches, 2)
	assert.Contains(t, parsed.Matches[0].Content, "hello")
}

func TestFileSearchExecute_FilesSearch(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app_config.yaml"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte(""), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "config",
		"path":    tmpDir,
		"target":  "files",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileSearchResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, "files", parsed.Target)
	assert.Equal(t, 1, parsed.Total)
	assert.Contains(t, parsed.Matches[0].File, "app_config.yaml")
}

func TestFileSearchExecute_EmptyPattern(t *testing.T) {
	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "",
	})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "pattern")
}

func TestFileSearchExecute_InvalidRegex(t *testing.T) {
	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "[invalid",
	})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "正则表达式")
}

func TestFileSearchExecute_InvalidTarget(t *testing.T) {
	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "test",
		"target":  "invalid",
	})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "不支持的 target")
}

func TestFileSearchExecute_NonexistentPath(t *testing.T) {
	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "test",
		"path":    "/nonexistent/dir",
	})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "搜索路径不存在")
}

func TestFileSearchExecute_DefaultTargetIsContent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("findme\n"), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "findme",
		"path":    tmpDir,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileSearchResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, "content", parsed.Target)
}

func TestFileSearchExecute_WithFileGlob(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.go"), []byte("findme in go\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "app.py"), []byte("findme in python\n"), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern":   "findme",
		"path":      tmpDir,
		"file_glob": "*.go",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileSearchResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, 1, parsed.Total)
	assert.Contains(t, parsed.Matches[0].File, "app.go")
}

func TestFileSearchExecute_SingleFileContentSearch(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "single.txt")
	require.NoError(t, os.WriteFile(fp, []byte("match1\nno hit\nmatch2\n"), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "match",
		"path":    fp,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileSearchResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, 2, parsed.Total)
}

func TestFileSearchExecute_SingleFileFileNameSearch(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "myapp_config.yaml")
	require.NoError(t, os.WriteFile(fp, []byte(""), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "config",
		"path":    fp,
		"target":  "files",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileSearchResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, 1, parsed.Total)
	assert.Contains(t, parsed.Matches[0].File, "myapp_config.yaml")
}

func TestFileSearchExecute_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("foo bar\n"), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "notfound",
		"path":    tmpDir,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileSearchResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, 0, parsed.Total)
	assert.Empty(t, parsed.Matches)
}

func TestFileSearchExecute_SummaryFormat(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("findme\n"), 0644))

	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "findme",
		"path":    tmpDir,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.Summary, "搜索完成")
	assert.Contains(t, result.Summary, "findme")
	assert.Contains(t, result.Summary, "1 个匹配")
}

func TestFileSearchExecute_DefaultPathIsCWD(t *testing.T) {
	// When no path is given, default is "." which gets expanded to cwd
	// This should at least not error
	tool := NewFileSearchTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"pattern": "nonexistent_pattern_xyz_12345",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}
