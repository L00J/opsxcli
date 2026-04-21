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

// --- formatSize ---

func TestFormatSize_Bytes(t *testing.T) {
	assert.Equal(t, "0B", formatSize(0))
	assert.Equal(t, "512B", formatSize(512))
}

func TestFormatSize_KB(t *testing.T) {
	assert.Equal(t, "1.0K", formatSize(1024))
}

func TestFormatSize_MB(t *testing.T) {
	assert.Equal(t, "1.0M", formatSize(1024*1024))
}

func TestFormatSize_GB(t *testing.T) {
	assert.Equal(t, "1.0G", formatSize(1024*1024*1024))
}

func TestFormatSize_LargeKB(t *testing.T) {
	assert.Equal(t, "512.0K", formatSize(512*1024))
}

// --- Cp ---

func TestCp_File(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	err := os.WriteFile(src, []byte("hello world"), 0644)
	require.NoError(t, err)

	err = Cp(src, dst, false)
	assert.NoError(t, err)

	data, err := os.ReadFile(dst)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestCp_FileNotFound(t *testing.T) {
	err := Cp("/nonexistent/file.txt", "/tmp/dst.txt", false)
	assert.Error(t, err)
}

func TestCp_DirRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srcdir")
	dstDir := filepath.Join(tmpDir, "dstdir")

	require.NoError(t, os.Mkdir(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0644))

	err := Cp(srcDir, dstDir, true)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dstDir, "a.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "aaa", string(data))
}

func TestCp_DirWithoutRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srcdir")
	require.NoError(t, os.Mkdir(srcDir, 0755))

	err := Cp(srcDir, filepath.Join(tmpDir, "dstdir"), false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "-r not specified")
}

// --- Mv ---

func TestMv_File(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("move me"), 0644))

	err := Mv(src, dst)
	assert.NoError(t, err)

	// Source should be gone
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))

	// Destination should have content
	data, err := os.ReadFile(dst)
	assert.NoError(t, err)
	assert.Equal(t, "move me", string(data))
}

// --- Rm ---

func TestRm_File(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "rmfile.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	err := Rm([]string{file}, false, false)
	assert.NoError(t, err)
	_, err = os.Stat(file)
	assert.True(t, os.IsNotExist(err))
}

func TestRm_Nonexistent_Force(t *testing.T) {
	err := Rm([]string{"/nonexistent/file.txt"}, false, true)
	assert.NoError(t, err) // force should suppress errors
}

func TestRm_Directory_Recursive(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "a.txt"), []byte("a"), 0644))

	err := Rm([]string{subDir}, true, false)
	assert.NoError(t, err)
	_, err = os.Stat(subDir)
	assert.True(t, os.IsNotExist(err))
}

func TestRm_Directory_NotRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))

	err := Rm([]string{subDir}, false, false)
	assert.NoError(t, err) // Rm returns nil but prints error to stderr
	// Directory should still exist
	_, err = os.Stat(subDir)
	assert.NoError(t, err)
}

// --- Mkdir ---

func TestMkdir_Single(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "newdir")

	err := Mkdir([]string{newDir}, false, 0755)
	assert.NoError(t, err)

	info, err := os.Stat(newDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestMkdir_Parents(t *testing.T) {
	tmpDir := t.TempDir()
	deepDir := filepath.Join(tmpDir, "a", "b", "c")

	err := Mkdir([]string{deepDir}, true, 0755)
	assert.NoError(t, err)

	info, err := os.Stat(deepDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- Touch ---

func TestTouch_CreateNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "newfile.txt")

	err := Touch([]string{file})
	assert.NoError(t, err)

	info, err := os.Stat(file)
	assert.NoError(t, err)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(0), info.Size())
}

func TestTouch_UpdateExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "existing.txt")
	require.NoError(t, os.WriteFile(file, []byte("data"), 0644))

	oldInfo, _ := os.Stat(file)
	err := Touch([]string{file})
	assert.NoError(t, err)

	newInfo, _ := os.Stat(file)
	// Mod time should be >= old mod time
	assert.True(t, newInfo.ModTime().After(oldInfo.ModTime()) || newInfo.ModTime().Equal(oldInfo.ModTime()))
	// Content should be preserved
	data, err := os.ReadFile(file)
	assert.NoError(t, err)
	assert.Equal(t, "data", string(data))
}

// --- Rmdir ---

func TestRmdir_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.Mkdir(emptyDir, 0755))

	err := Rmdir([]string{emptyDir}, false)
	assert.NoError(t, err)
	_, err = os.Stat(emptyDir)
	assert.True(t, os.IsNotExist(err))
}

// --- Ln ---

func TestLn_Symbolic(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	link := filepath.Join(tmpDir, "link.txt")

	require.NoError(t, os.WriteFile(target, []byte("linked"), 0644))

	err := Ln(target, link, true)
	assert.NoError(t, err)

	data, err := os.ReadFile(link)
	assert.NoError(t, err)
	assert.Equal(t, "linked", string(data))
}

func TestLn_Hard(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	link := filepath.Join(tmpDir, "link.txt")

	require.NoError(t, os.WriteFile(target, []byte("hardlinked"), 0644))

	err := Ln(target, link, false)
	assert.NoError(t, err)

	data, err := os.ReadFile(link)
	assert.NoError(t, err)
	assert.Equal(t, "hardlinked", string(data))
}

// --- Dd ---

func TestDd_CopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.bin")
	dst := filepath.Join(tmpDir, "dst.bin")

	// Create a 1KB source file
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	require.NoError(t, os.WriteFile(src, data, 0644))

	err := Dd(DdOptions{
		If:    src,
		Of:    dst,
		Bs:    512,
		Count: 2,
	})
	assert.NoError(t, err)

	result, err := os.ReadFile(dst)
	assert.NoError(t, err)
	assert.Equal(t, 1024, len(result))
}

func TestDd_InputNotFound(t *testing.T) {
	err := Dd(DdOptions{
		If: "/nonexistent/file.bin",
		Of: "/tmp/output.bin",
		Bs: 512,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "打开输入文件失败")
}

func TestDd_DefaultBlockSize(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.bin")
	dst := filepath.Join(tmpDir, "dst.bin")

	// Create small file
	require.NoError(t, os.WriteFile(src, []byte("hello"), 0644))

	err := Dd(DdOptions{
		If: src,
		Of: dst,
		// Bs intentionally 0, should default to 512
	})
	assert.NoError(t, err)

	result, err := os.ReadFile(dst)
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(result))
}

// --- Grep ---

func TestGrep_BasicMatch(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello world\nfoo bar\nhello again\n"), 0644))

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Grep("hello", []string{file}, GrepOptions{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "hello world")
	assert.Contains(t, output, "hello again")
	assert.NotContains(t, output, "foo bar")
}

func TestGrep_InvertMatch(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello world\nfoo bar\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Grep("hello", []string{file}, GrepOptions{InvertMatch: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.NotContains(t, output, "hello world")
	assert.Contains(t, output, "foo bar")
}

func TestGrep_IgnoreCase(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("Hello World\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Grep("hello", []string{file}, GrepOptions{IgnoreCase: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Hello World")
}

// --- DdOptions defaults ---

func TestDdOptions_Defaults(t *testing.T) {
	opts := DdOptions{}
	assert.Equal(t, int64(0), opts.Bs)
	assert.Equal(t, "", opts.Status)
	assert.Equal(t, int64(0), opts.Count)
}

// --- LsOptions ---

func TestLsOptions_Fields(t *testing.T) {
	opts := LsOptions{All: true, Long: true, Human: true}
	assert.True(t, opts.All)
	assert.True(t, opts.Long)
	assert.True(t, opts.Human)
	assert.False(t, opts.Recursive)
}

// --- GrepOptions ---

func TestGrepOptions_Fields(t *testing.T) {
	opts := GrepOptions{IgnoreCase: true, LineNumber: true, CountOnly: true}
	assert.True(t, opts.IgnoreCase)
	assert.True(t, opts.LineNumber)
	assert.True(t, opts.CountOnly)
	assert.False(t, opts.InvertMatch)
}

// --- Cat (basic) ---

func TestCat_FileNotFound(t *testing.T) {
	// Cat should not return error for missing files (prints to stderr)
	err := Cat([]string{"/nonexistent/file.txt"}, false)
	assert.NoError(t, err)
}

// --- Head ---

func TestHead_DefaultLines(t *testing.T) {
	// Test that lines defaults to 10
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, strings.Repeat("x", 5))
	}
	require.NoError(t, os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Head([]string{file}, 0) // 0 should default to 10

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	lineCount := strings.Count(strings.TrimSpace(output), "\n") + 1
	assert.Equal(t, 10, lineCount)
}
