package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== tree 测试 ====================

func TestTree_BasicOutput(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "subdir", "b.txt"), []byte("b"), 0644))

	// 捕获 stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Tree([]string{tmpDir}, TreeOptions{})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	assert.Contains(t, output, "a.txt")
	assert.Contains(t, output, "subdir/")
	assert.Contains(t, output, "b.txt")
	assert.Contains(t, output, "directories")
}

func TestTree_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "a", "b", "c"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a", "b", "c", "deep.txt"), []byte("x"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Tree([]string{tmpDir}, TreeOptions{MaxDepth: 1})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	// 深度 1 应该只显示 a/ 目录，不显示 a/b/c/deep.txt
	assert.Contains(t, output, "a/")
	assert.NotContains(t, output, "deep.txt")
}

func TestTree_DirsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "mydir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("f"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Tree([]string{tmpDir}, TreeOptions{DirsOnly: true})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	assert.Contains(t, output, "mydir/")
	assert.NotContains(t, output, "file.txt")
}

func TestTree_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("h"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("v"), 0644))

	// 不显示隐藏文件
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	Tree([]string{tmpDir}, TreeOptions{})
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	assert.NotContains(t, buf.String(), ".hidden")
	assert.Contains(t, buf.String(), "visible.txt")

	// 显示隐藏文件
	r, w, _ = os.Pipe()
	os.Stdout = w
	Tree([]string{tmpDir}, TreeOptions{All: true})
	w.Close()
	os.Stdout = old
	buf.Reset()
	buf.ReadFrom(r)
	assert.Contains(t, buf.String(), ".hidden")
}

func TestTree_NotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	err := Tree([]string{file}, TreeOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不是目录")
}

func TestTree_DefaultPath(t *testing.T) {
	// 默认使用当前目录 "."
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Tree([]string{}, TreeOptions{})

	w.Close()
	os.Stdout = old

	// 消费 pipe 数据避免阻塞
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err)
}

// ==================== du 测试 ====================

func TestDu_DefaultPath(t *testing.T) {
	// 默认使用当前目录
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Du([]string{}, DuOptions{})

	w.Close()
	os.Stdout = old

	// 消费 pipe 数据
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err)
}

func TestDu_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte(strings.Repeat("x", 1024)), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Du([]string{file}, DuOptions{Human: true})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	assert.Contains(t, output, "test.txt")
}

func TestDu_Summarize(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("bbb"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Du([]string{tmpDir}, DuOptions{Summarize: true, Human: true})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	// 总结模式只显示一行
	lines := strings.Count(strings.TrimSpace(output), "\n") + 1
	assert.Equal(t, 1, lines)
}

func TestDu_NonExistentPath(t *testing.T) {
	err := Du([]string{"/nonexistent/path"}, DuOptions{})
	assert.Error(t, err)
}

// ==================== humanSize 测试 ====================

func TestHumanSize_Units(t *testing.T) {
	assert.Equal(t, "512B", humanSize(512))
	assert.Equal(t, "1.0Ki", humanSize(1024))
	assert.Equal(t, "1.0Mi", humanSize(1024*1024))
	assert.Equal(t, "1.0Gi", humanSize(1024*1024*1024))
	assert.Equal(t, "1.0Ti", humanSize(1024*1024*1024*1024))
}

func TestHumanSize_Zero(t *testing.T) {
	assert.Equal(t, "0B", humanSize(0))
}

func TestHumanSize_LargeValues(t *testing.T) {
	assert.Equal(t, "1.5Ki", humanSize(1536))
	assert.Equal(t, "2.5Mi", humanSize(2621440))
}
