package tools

import (
	"context"

	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchFileContent_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world\nfoo bar\nhello again\n"
	err := os.WriteFile(testFile, []byte(content), 0644)
	assert.NoError(t, err)

	info, err := os.Stat(testFile)
	assert.NoError(t, err)

	re := regexp.MustCompile(`hello`)
	matches, truncated, err := searchFileContent(context.Background(), re, testFile, info, "")
	assert.NoError(t, err)
	assert.False(t, truncated)
	assert.Len(t, matches, 2)
	assert.Contains(t, matches[0].Content, "hello")
	assert.Equal(t, 1, matches[0].Line)
	assert.Equal(t, 3, matches[1].Line)
}

func TestSearchFileContent_SingleFileWithGlob(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\nfunc main() {}\n"), 0644)

	info, _ := os.Stat(testFile)
	re := regexp.MustCompile(`main`)

	matches, _, err := searchFileContent(context.Background(), re, testFile, info, "*.go")
	assert.NoError(t, err)
	assert.Len(t, matches, 2)

	matches2, _, err := searchFileContent(context.Background(), re, testFile, info, "*.txt")
	assert.NoError(t, err)
	assert.Len(t, matches2, 0)
}

func TestSearchFileContent_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("findme in a\nother line\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("no match here\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "c.txt"), []byte("findme in c\n"), 0644)

	info, _ := os.Stat(tmpDir)
	re := regexp.MustCompile(`findme`)

	matches, truncated, err := searchFileContent(context.Background(), re, tmpDir, info, "")
	assert.NoError(t, err)
	assert.False(t, truncated)
	assert.Len(t, matches, 2)
}

func TestSearchFileContent_SkipsBinaryFiles(t *testing.T) {
	tmpDir := t.TempDir()
	binFile := filepath.Join(tmpDir, "binary.bin")
	os.WriteFile(binFile, []byte{0x00, 0x01, 0x02, 0xFF}, 0644)

	txtFile := filepath.Join(tmpDir, "text.txt")
	os.WriteFile(txtFile, []byte("hello world\n"), 0644)

	info, _ := os.Stat(tmpDir)
	re := regexp.MustCompile(`hello`)

	matches, _, err := searchFileContent(context.Background(), re, tmpDir, info, "")
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Contains(t, matches[0].File, "text.txt")
}

func TestSearchFileContent_SkipsHiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("findme hidden\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("findme visible\n"), 0644)

	info, _ := os.Stat(tmpDir)
	re := regexp.MustCompile(`findme`)

	matches, _, err := searchFileContent(context.Background(), re, tmpDir, info, "")
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Contains(t, matches[0].File, "visible.txt")
}

func TestSearchFileContent_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("foo bar\nbaz qux\n"), 0644)

	info, _ := os.Stat(testFile)
	re := regexp.MustCompile(`notfound`)

	matches, truncated, err := searchFileContent(context.Background(), re, testFile, info, "")
	assert.NoError(t, err)
	assert.False(t, truncated)
	assert.Len(t, matches, 0)
}

func TestSearchFileNames_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "myapp_config.yaml")
	os.WriteFile(testFile, []byte("key: value\n"), 0644)

	info, _ := os.Stat(testFile)
	re := regexp.MustCompile(`config`)

	// fileGlob 为空时，单文件名匹配 regex 应成功
	matches, truncated, err := searchFileNames(re, testFile, info, "")
	assert.NoError(t, err)
	assert.False(t, truncated)
	assert.Len(t, matches, 1)
}

func TestSearchFileNames_SingleFileNotMatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readme.md")
	os.WriteFile(testFile, []byte("hello\n"), 0644)

	info, _ := os.Stat(testFile)
	re := regexp.MustCompile(`config`)

	matches, _, err := searchFileNames(re, testFile, info, "")
	assert.NoError(t, err)
	assert.Len(t, matches, 0)
}

func TestSearchFileNames_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "app_config.yaml"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "db_config.json"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "readme.md"), []byte(""), 0644)

	info, _ := os.Stat(tmpDir)
	re := regexp.MustCompile(`config`)

	matches, truncated, err := searchFileNames(re, tmpDir, info, "")
	assert.NoError(t, err)
	assert.False(t, truncated)
	assert.Len(t, matches, 2)
}

func TestSearchFileNames_WithGlob(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "app_config.yaml"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "app_config.json"), []byte(""), 0644)

	info, _ := os.Stat(tmpDir)
	re := regexp.MustCompile(`config`)

	matches, _, err := searchFileNames(re, tmpDir, info, "*.yaml")
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Contains(t, matches[0].File, "app_config.yaml")
}

func TestSearchFileNames_SkipsHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, ".git"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "config_file"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "config_file"), []byte(""), 0644)

	info, _ := os.Stat(tmpDir)
	re := regexp.MustCompile(`config`)

	matches, _, err := searchFileNames(re, tmpDir, info, "")
	assert.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.NotContains(t, matches[0].File, ".git")
}
