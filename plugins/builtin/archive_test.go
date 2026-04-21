package builtin

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== gzip 测试 ====================

func TestGzip_NoFiles(t *testing.T) {
	err := Gzip([]string{}, GzipOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要指定文件")
}

func TestGzip_CompressAndDecompress(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(src, []byte("hello gzip world"), 0644))

	// 压缩
	err := Gzip([]string{src}, GzipOptions{})
	assert.NoError(t, err)

	gzFile := src + ".gz"
	_, err = os.Stat(gzFile)
	assert.NoError(t, err, "应该创建 .gz 文件")

	// 原文件应该被删除（默认行为）
	_, err = os.Stat(src)
	assert.True(t, os.IsNotExist(err), "原文件应该被删除")

	// 解压
	err = Gzip([]string{gzFile}, GzipOptions{Decompress: true})
	assert.NoError(t, err)

	data, err := os.ReadFile(src)
	assert.NoError(t, err)
	assert.Equal(t, "hello gzip world", string(data))
}

func TestGzip_KeepOriginal(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "keep.txt")
	require.NoError(t, os.WriteFile(src, []byte("keep me"), 0644))

	err := Gzip([]string{src}, GzipOptions{Keep: true})
	assert.NoError(t, err)

	// 原文件应该保留
	_, err = os.Stat(src)
	assert.NoError(t, err, "原文件应该保留")

	// .gz 文件也存在
	_, err = os.Stat(src + ".gz")
	assert.NoError(t, err)
}

func TestGzip_Stdout(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "stdout.txt")
	require.NoError(t, os.WriteFile(src, []byte("stdout content"), 0644))

	// 捕获 stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Gzip([]string{src}, GzipOptions{Stdout: true, Keep: true})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)

	// 输出应该是 gzip 格式
	gr, err := gzip.NewReader(&buf)
	assert.NoError(t, err)
	decompressed, _ := io.ReadAll(gr)
	gr.Close()
	assert.Equal(t, "stdout content", string(decompressed))

	// 原文件应该保留（Keep 未设置时不删除因为 Stdout 模式）
	_, err = os.Stat(src)
	assert.NoError(t, err)
}

func TestGzip_NonExistentFile(t *testing.T) {
	err := Gzip([]string{"/nonexistent/file.txt"}, GzipOptions{})
	assert.Error(t, err)
}

func TestGunzip_InvalidName(t *testing.T) {
	tmpDir := t.TempDir()
	// 创建一个没有 .gz 后缀的文件
	src := filepath.Join(tmpDir, "notaarchive")
	require.NoError(t, os.WriteFile(src, []byte("not gzipped"), 0644))

	err := Gzip([]string{src}, GzipOptions{Decompress: true})
	// gunzipFile 先尝试 gzip 解码，非 gzip 内容会失败
	assert.Error(t, err)
}

// ==================== unzip 测试 ====================

func TestUnzip_NoFiles(t *testing.T) {
	err := Unzip([]string{}, UnzipOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要指定")
}

func TestUnzip_ListContents(t *testing.T) {
	tmpDir := t.TempDir()
	zipFile := filepath.Join(tmpDir, "test.zip")

	// 创建测试 ZIP 文件
	createTestZip(t, zipFile, map[string]string{
		"hello.txt": "hello world",
		"subdir/":   "",
	})

	// 捕获 stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Unzip([]string{zipFile}, UnzipOptions{List: true})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	assert.Contains(t, output, "hello.txt")
}

func TestUnzip_Extract(t *testing.T) {
	tmpDir := t.TempDir()
	zipFile := filepath.Join(tmpDir, "test.zip")
	extractDir := filepath.Join(tmpDir, "extracted")

	// 创建测试 ZIP 文件
	createTestZip(t, zipFile, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	})

	require.NoError(t, os.MkdirAll(extractDir, 0755))

	err := Unzip([]string{zipFile}, UnzipOptions{Dir: extractDir})
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(extractDir, "file1.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "content1", string(data))

	data, err = os.ReadFile(filepath.Join(extractDir, "file2.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "content2", string(data))
}

func TestUnzip_Quiet(t *testing.T) {
	tmpDir := t.TempDir()
	zipFile := filepath.Join(tmpDir, "quiet.zip")
	extractDir := filepath.Join(tmpDir, "qout")

	createTestZip(t, zipFile, map[string]string{
		"q.txt": "quiet",
	})
	require.NoError(t, os.MkdirAll(extractDir, 0755))

	// 捕获 stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Unzip([]string{zipFile}, UnzipOptions{Dir: extractDir, Quiet: true})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	// 静默模式不应该输出 extracting 信息
	assert.NotContains(t, buf.String(), "extracting")
}

func TestUnzip_NonExistentFile(t *testing.T) {
	err := Unzip([]string{"/nonexistent/file.zip"}, UnzipOptions{})
	assert.Error(t, err)
}

// ==================== tar 测试 ====================

func TestTar_NoOperation(t *testing.T) {
	err := Tar([]string{}, TarOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要指定操作")
}

func TestTar_CreateAndExtract(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "b.txt"), []byte("bbb"), 0644))

	tarFile := filepath.Join(tmpDir, "test.tar")

	// 创建 tar
	err := Tar([]string{srcDir}, TarOptions{
		Create: true,
		File:   tarFile,
	})
	assert.NoError(t, err)

	_, err = os.Stat(tarFile)
	assert.NoError(t, err, "应该创建 tar 文件")

	// 解压 tar
	extractDir := filepath.Join(tmpDir, "extracted")
	require.NoError(t, os.MkdirAll(extractDir, 0755))

	err = Tar([]string{}, TarOptions{
		Extract:   true,
		File:      tarFile,
		Directory: extractDir,
	})
	assert.NoError(t, err)

	// 验证解压结果
	data, err := os.ReadFile(filepath.Join(extractDir, srcDir, "a.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "aaa", string(data))
}

func TestTar_CreateWithGzip(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srcgz")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "g.txt"), []byte("gzipped"), 0644))

	tarFile := filepath.Join(tmpDir, "test.tar.gz")

	err := Tar([]string{srcDir}, TarOptions{
		Create: true,
		File:   tarFile,
		Gzip:   true,
	})
	assert.NoError(t, err)

	// 验证文件存在
	_, err = os.Stat(tarFile)
	assert.NoError(t, err)

	// 验证文件确实是 gzip 格式
	f, err := os.Open(tarFile)
	require.NoError(t, err)
	defer f.Close()

	buf := make([]byte, 2)
	_, err = f.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1f, 0x8b}, buf, "应该是 gzip 魔术头")
}

func TestTar_ListContents(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srclist")
	require.NoError(t, os.MkdirAll(srcDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "listed.txt"), []byte("listed"), 0644))

	tarFile := filepath.Join(tmpDir, "list.tar")

	// 先创建
	err := Tar([]string{srcDir}, TarOptions{Create: true, File: tarFile})
	require.NoError(t, err)

	// 捕获 stdout 列出内容
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = Tar([]string{}, TarOptions{List: true, File: tarFile})

	w.Close()
	os.Stdout = old

	assert.NoError(t, err)

	var buf bytes.Buffer
	buf.ReadFrom(r)
	assert.Contains(t, buf.String(), "listed.txt")
}

func TestTar_NoFile(t *testing.T) {
	err := Tar([]string{}, TarOptions{Create: true})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "归档文件名")
}

// ==================== 辅助函数 ====================

// createTestZip 创建一个测试 ZIP 文件
func createTestZip(t *testing.T, zipPath string, files map[string]string) {
	t.Helper()
	f, err := os.Create(zipPath)
	require.NoError(t, err)
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		if name[len(name)-1] == '/' {
			// 目录
			_, err := w.Create(name)
			require.NoError(t, err)
		} else {
			fw, err := w.Create(name)
			require.NoError(t, err)
			_, err = fw.Write([]byte(content))
			require.NoError(t, err)
		}
	}
	require.NoError(t, w.Close())
}
