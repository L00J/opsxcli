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
// FileReadTool.Execute integration tests
// ---------------------------------------------------------------------------

func TestFileReadExecute_BasicTextFile(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "hello.txt")
	content := "line one\nline two\nline three\n"
	require.NoError(t, os.WriteFile(fp, []byte(content), 0644))

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": fp,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	// Parse JSON output
	var parsed FileReadResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, fp, parsed.Path)
	assert.Equal(t, 3, parsed.TotalLines)
	assert.Len(t, parsed.Lines, 3)
	assert.Equal(t, "line one", parsed.Lines[0].Content)
	assert.Equal(t, 1, parsed.Lines[0].Num)
	assert.Equal(t, "line three", parsed.Lines[2].Content)
	assert.False(t, parsed.Truncated)
	assert.Contains(t, result.Summary, "hello.txt")
}

func TestFileReadExecute_WithOffsetAndLimit(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "lines.txt")
	var content string
	for i := 1; i <= 20; i++ {
		content += "line\n"
	}
	require.NoError(t, os.WriteFile(fp, []byte(content), 0644))

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":   fp,
		"offset": float64(5),
		"limit":  float64(3),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileReadResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, 20, parsed.TotalLines)
	assert.Len(t, parsed.Lines, 3)
	assert.Equal(t, 5, parsed.Lines[0].Num)
	assert.Equal(t, 7, parsed.Lines[2].Num)
	assert.True(t, parsed.Truncated)
}

func TestFileReadExecute_EmptyPath(t *testing.T) {
	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "",
	})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path")
}

func TestFileReadExecute_NonexistentFile(t *testing.T) {
	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "/nonexistent/path/to/file.txt",
	})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "文件不存在")
}

func TestFileReadExecute_PathIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpDir,
	})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "目录")
}

func TestFileReadExecute_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "binary.dat")
	data := make([]byte, 100)
	data[0] = 0x00 // null byte → binary
	require.NoError(t, os.WriteFile(fp, data, 0644))

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": fp,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "二进制文件")
	assert.Contains(t, result.Summary, "二进制文件")
}

func TestFileReadExecute_TildePath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	// Create a temp file in home dir
	fp := filepath.Join(home, ".opsx_test_read_tmp")
	require.NoError(t, os.WriteFile(fp, []byte("test content\n"), 0644))
	defer os.Remove(fp)

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": "~/.opsx_test_read_tmp",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileReadResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, fp, parsed.Path)
	assert.Equal(t, 1, parsed.TotalLines)
	assert.Equal(t, "test content", parsed.Lines[0].Content)
}

func TestFileReadExecute_LimitCappedAt2000(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "big.txt")
	require.NoError(t, os.WriteFile(fp, []byte("line\n"), 0644))

	tool := NewFileReadTool()
	// Pass limit > 2000, should be capped
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":  fp,
		"limit": float64(9999),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileReadResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	// Only 1 line in file; limit was capped but that doesn't affect a 1-line file
	assert.Equal(t, 1, parsed.TotalLines)
	assert.Len(t, parsed.Lines, 1)
}

func TestFileReadExecute_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "empty.txt")
	require.NoError(t, os.WriteFile(fp, []byte(""), 0644))

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": fp,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileReadResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, 0, parsed.TotalLines)
	assert.Empty(t, parsed.Lines)
}

func TestFileReadExecute_OffsetBeyondFile(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "short.txt")
	require.NoError(t, os.WriteFile(fp, []byte("only line\n"), 0644))

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":   fp,
		"offset": float64(100),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)

	var parsed FileReadResult
	require.NoError(t, json.Unmarshal([]byte(result.Output), &parsed))
	assert.Equal(t, 1, parsed.TotalLines)
	assert.Empty(t, parsed.Lines)
}

func TestFileReadExecute_InvalidRegexCharsInPath(t *testing.T) {
	tmpDir := t.TempDir()
	fp := filepath.Join(tmpDir, "file[1].txt")
	require.NoError(t, os.WriteFile(fp, []byte("content\n"), 0644))

	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": fp,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestFileReadExecute_NoPathParam(t *testing.T) {
	tool := NewFileReadTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
}
