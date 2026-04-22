package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== formatSize 额外测试 ====================

func TestFormatSize_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"one_byte", 1, "1B"},
		{"near_1k", 1023, "1023B"},
		{"exactly_1k", 1024, "1.0K"},
		{"1point5k", 1536, "1.5K"},
		{"near_1m", 1024*1024 - 1, "1024.0K"},
		{"exactly_1m", 1024 * 1024, "1.0M"},
		{"1point5m", 1536 * 1024, "1.5M"},
		{"exactly_1g", 1024 * 1024 * 1024, "1.0G"},
		{"2g", 2 * 1024 * 1024 * 1024, "2.0G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatSize(tt.input))
		})
	}
}

func TestFormatSize_TB(t *testing.T) {
	input := int64(1024) * 1024 * 1024 * 1024
	assert.Equal(t, "1.0T", formatSize(input))
}

// ==================== Cat 额外测试 ====================

func TestCat_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "hello.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello world\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Cat([]string{file}, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "hello world")
}

func TestCat_WithLineNumbers(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "lines.txt")
	require.NoError(t, os.WriteFile(file, []byte("line1\nline2\nline3\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Cat([]string{file}, true)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "line1")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "line2")
	assert.Contains(t, output, "3")
	assert.Contains(t, output, "line3")
}

func TestCat_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "a.txt")
	file2 := filepath.Join(tmpDir, "b.txt")
	require.NoError(t, os.WriteFile(file1, []byte("AAA\n"), 0644))
	require.NoError(t, os.WriteFile(file2, []byte("BBB\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Cat([]string{file1, file2}, false)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "AAA")
	assert.Contains(t, output, "BBB")
}

// ==================== Grep 额外测试 ====================

func TestGrep_CountOnly(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("hello world\nfoo bar\nhello again\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Grep("hello", []string{file}, GrepOptions{CountOnly: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "2")
}

func TestGrep_LineNumber(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("aaa\nbbb\naaa\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Grep("aaa", []string{file}, GrepOptions{LineNumber: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "1:")
	assert.Contains(t, output, "3:")
}

func TestGrep_WithFilename(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("match here\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Grep("match", []string{file}, GrepOptions{WithFilename: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "test.txt:")
}

func TestGrep_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(file, []byte("aaa\nbbb\nccc\n"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Grep("zzz", []string{file}, GrepOptions{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Equal(t, "", output)
}

// ==================== Ls 测试 ====================

func TestLs_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "single.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Ls([]string{file}, LsOptions{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "single.txt")
}

func TestLs_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Ls([]string{tmpDir}, LsOptions{})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "a.txt")
	assert.Contains(t, output, "b.txt")
}

func TestLs_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("h"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "visible.txt"), []byte("v"), 0644))

	// 不显示隐藏文件
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	Ls([]string{tmpDir}, LsOptions{})
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	assert.NotContains(t, buf.String(), ".hidden")
	assert.Contains(t, buf.String(), "visible.txt")

	// 显示隐藏文件
	r, w, _ = os.Pipe()
	os.Stdout = w
	Ls([]string{tmpDir}, LsOptions{All: true})
	w.Close()
	os.Stdout = old
	buf.Reset()
	buf.ReadFrom(r)
	assert.Contains(t, buf.String(), ".hidden")
	assert.Contains(t, buf.String(), "visible.txt")
}

func TestLs_LongFormat(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Ls([]string{tmpDir}, LsOptions{Long: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "file.txt")
	// Long format should contain permission info
	assert.Contains(t, output, "-rw-")
}

func TestLs_LongFormatHuman(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("x"), 0644))

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Ls([]string{tmpDir}, LsOptions{Long: true, Human: true})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "file.txt")
	assert.Contains(t, output, "B")
}

func TestLs_DefaultPath(t *testing.T) {
	// 空路径应该默认为当前目录
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Ls([]string{}, LsOptions{})

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err)
}

func TestLs_NonExistentPath(t *testing.T) {
	// 不存在的路径应该打印错误但不返回 error
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := Ls([]string{"/nonexistent/path"}, LsOptions{})

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err) // Ls returns nil even for errors
	assert.Contains(t, buf.String(), "nonexistent")
}

// ==================== Rmdir 额外测试 ====================

func TestRmdir_Parents(t *testing.T) {
	tmpDir := t.TempDir()
	deepDir := filepath.Join(tmpDir, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deepDir, 0755))

	err := Rmdir([]string{deepDir}, true)
	assert.NoError(t, err)

	// All directories up the chain should be removed
	_, err = os.Stat(filepath.Join(tmpDir, "a"))
	assert.True(t, os.IsNotExist(err))
}

func TestRmdir_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	nonExist := filepath.Join(tmpDir, "nope")

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := Rmdir([]string{nonExist}, false)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "目录不存在")
}

func TestRmdir_NonEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("x"), 0644))

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := Rmdir([]string{subDir}, false)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err) // Rmdir returns nil but prints error
	assert.Contains(t, buf.String(), "directory not empty")
}

// ==================== Chmod 测试 ====================

func TestChmod_BasicFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "chmod_test.txt")
	require.NoError(t, os.WriteFile(file, []byte("chmod me"), 0644))

	err := Chmod("755", []string{file}, false)
	assert.NoError(t, err)

	info, err := os.Stat(file)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestChmod_InvalidMode(t *testing.T) {
	err := Chmod("abc", []string{"/tmp/somefile"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的权限模式")
}

func TestChmod_Recursive(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	file := filepath.Join(subDir, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0644))

	err := Chmod("755", []string{tmpDir}, true)
	assert.NoError(t, err)

	// Subdirectory file should also be chmodded
	info, err := os.Stat(file)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

// ==================== Touch 额外测试 ====================

func TestTouch_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")

	err := Touch([]string{f1, f2})
	assert.NoError(t, err)

	_, err = os.Stat(f1)
	assert.NoError(t, err)
	_, err = os.Stat(f2)
	assert.NoError(t, err)
}

// ==================== Mkdir 额外测试 ====================

func TestMkdir_MultiplePaths(t *testing.T) {
	tmpDir := t.TempDir()
	d1 := filepath.Join(tmpDir, "dir1")
	d2 := filepath.Join(tmpDir, "dir2")

	err := Mkdir([]string{d1, d2}, false, 0755)
	assert.NoError(t, err)

	info, err := os.Stat(d1)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	info, err = os.Stat(d2)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestMkdir_ExistingDir(t *testing.T) {
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "exists")
	require.NoError(t, os.Mkdir(existingDir, 0755))

	// Mkdir on existing dir prints to stderr but returns nil
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := Mkdir([]string{existingDir}, false, 0755)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "exists")
}

// ==================== Rm 额外测试 ====================

func TestRm_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	require.NoError(t, os.WriteFile(f1, []byte("a"), 0644))
	require.NoError(t, os.WriteFile(f2, []byte("b"), 0644))

	err := Rm([]string{f1, f2}, false, false)
	assert.NoError(t, err)

	_, err = os.Stat(f1)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(f2)
	assert.True(t, os.IsNotExist(err))
}

func TestRm_NonExistent_NoForce(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	err := Rm([]string{"/nonexistent/file.txt"}, false, false)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "nonexistent")
}

// ==================== Cp 额外测试 ====================

func TestCp_PreservesPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	require.NoError(t, os.WriteFile(src, []byte("perm test"), 0755))

	err := Cp(src, dst, false)
	assert.NoError(t, err)

	info, err := os.Stat(dst)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestCp_NestedDir(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub", "deep.txt"), []byte("deep"), 0644))

	dstDir := filepath.Join(tmpDir, "dst")

	err := Cp(srcDir, dstDir, true)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dstDir, "sub", "deep.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "deep", string(data))
}

// ==================== Mv 额外测试 ====================

func TestMv_NonExistentSource(t *testing.T) {
	err := Mv("/nonexistent/src.txt", "/tmp/dst.txt")
	assert.Error(t, err)
}

func TestMv_IntoDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "file.txt")
	dstDir := filepath.Join(tmpDir, "target")
	require.NoError(t, os.WriteFile(src, []byte("move me"), 0644))
	require.NoError(t, os.Mkdir(dstDir, 0755))

	// Mv into directory - dst is the directory, file keeps same name
	dstFile := filepath.Join(dstDir, "file.txt")
	err := Mv(src, dstFile)
	assert.NoError(t, err)

	data, err := os.ReadFile(dstFile)
	assert.NoError(t, err)
	assert.Equal(t, "move me", string(data))

	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err))
}
