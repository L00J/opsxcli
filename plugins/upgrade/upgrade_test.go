package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- convertOSName ---

func TestConvertOSName_Linux(t *testing.T) {
	assert.Equal(t, "Linux", convertOSName("linux"))
}

func TestConvertOSName_Darwin(t *testing.T) {
	assert.Equal(t, "Darwin", convertOSName("darwin"))
}

func TestConvertOSName_Windows(t *testing.T) {
	assert.Equal(t, "Windows", convertOSName("windows"))
}

func TestConvertOSName_Empty(t *testing.T) {
	assert.Equal(t, "", convertOSName(""))
}

// --- convertArch ---

func TestConvertArch_Amd64(t *testing.T) {
	assert.Equal(t, "x86_64", convertArch("linux", "amd64"))
	assert.Equal(t, "x86_64", convertArch("darwin", "amd64"))
	assert.Equal(t, "x86_64", convertArch("windows", "amd64"))
}

func TestConvertArch_Arm64_Linux(t *testing.T) {
	assert.Equal(t, "aarch64", convertArch("linux", "arm64"))
}

func TestConvertArch_Arm64_Darwin(t *testing.T) {
	assert.Equal(t, "arm64", convertArch("darwin", "arm64"))
}

func TestConvertArch_386(t *testing.T) {
	assert.Equal(t, "i386", convertArch("linux", "386"))
}

func TestConvertArch_Unknown(t *testing.T) {
	assert.Equal(t, "mips", convertArch("linux", "mips"))
}

// --- detectInstallPath ---

func TestDetectInstallPath_StandardPath(t *testing.T) {
	// 当路径已经是标准路径时，直接返回
	assert.Equal(t, "/usr/local/bin/opsxcli", detectInstallPath("/usr/local/bin/opsxcli"))
	assert.Equal(t, "/usr/bin/opsxcli", detectInstallPath("/usr/bin/opsxcli"))
}

func TestDetectInstallPath_NonStandardPath(t *testing.T) {
	// 非标准路径时，返回当前路径或找到的标准路径
	result := detectInstallPath("/home/user/opsxcli")
	assert.NotEmpty(t, result)
}

// --- writeCounter ---

func TestWriteCounter_Write(t *testing.T) {
	// 测试 Write 方法正确累计下载字节数
	wc := &writeCounter{Total: 100}
	n, err := wc.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(5), wc.Downloaded)
}

func TestWriteCounter_Write多次调用(t *testing.T) {
	// 测试多次调用 Write 累计字节数
	wc := &writeCounter{Total: 100}
	wc.Write([]byte("hello"))  // 5
	wc.Write([]byte(" world")) // 6
	assert.Equal(t, int64(11), wc.Downloaded)
}

func TestWriteCounter_Write空字节(t *testing.T) {
	// 测试写入空切片
	wc := &writeCounter{Total: 100}
	n, err := wc.Write([]byte{})
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, int64(0), wc.Downloaded)
}

func TestWriteCounter_PrintProgress_Total为零(t *testing.T) {
	// 当 Total <= 0 时不应 panic
	wc := &writeCounter{Total: 0}
	wc.Downloaded = 50
	assert.NotPanics(t, func() {
		wc.printProgress()
	})
}

func TestWriteCounter_PrintProgress_Total为负(t *testing.T) {
	// 当 Total 为负数时不应 panic
	wc := &writeCounter{Total: -1}
	wc.Downloaded = 50
	assert.NotPanics(t, func() {
		wc.printProgress()
	})
}

func TestWriteCounter_PrintProgress_正常百分比(t *testing.T) {
	// 正常计算百分比不应 panic
	wc := &writeCounter{Total: 200}
	wc.Downloaded = 100
	assert.NotPanics(t, func() {
		wc.printProgress()
	})
}

// --- downloadFile ---

func TestDownloadFile_成功下载(t *testing.T) {
	// 使用 httptest mock HTTP 服务器，测试正常下载
	content := []byte("这是测试文件内容")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证 User-Agent 头
		assert.Equal(t, userAgent, r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "downloaded.txt")
	err := downloadFile(server.URL, dest)
	require.NoError(t, err)

	// 验证文件内容
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestDownloadFile_HTTP错误状态码(t *testing.T) {
	// 测试服务器返回非 200 状态码
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "downloaded.txt")
	err := downloadFile(server.URL, dest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestDownloadFile_无效URL(t *testing.T) {
	// 测试格式错误的 URL（快速失败，不依赖网络）
	dest := filepath.Join(t.TempDir(), "downloaded.txt")
	err := downloadFile("://invalid-url", dest)
	assert.Error(t, err)
}

func TestDownloadFile_无效目标路径(t *testing.T) {
	// 测试目标路径不存在（无法创建文件）
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("content"))
	}))
	defer server.Close()

	// 目标路径在一个不存在的目录中
	dest := filepath.Join(t.TempDir(), "nonexistent-dir", "file.txt")
	err := downloadFile(server.URL, dest)
	assert.Error(t, err)
}

// --- extractTarGz ---

// helper: 创建包含 opsxcli 二进制的 tar.gz 文件
func createTarGz(t *testing.T, dir string, fileName string, content string) string {
	t.Helper()
	tarGzPath := filepath.Join(dir, fileName)

	var buf bytes.Buffer
	// gzip writer
	gzw := gzip.NewWriter(&buf)
	// tar writer
	tw := tar.NewWriter(gzw)

	hdr := &tar.Header{
		Name: "opsxcli",
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))
	return tarGzPath
}

// helper: 创建包含子目录中 opsxcli 的 tar.gz 文件
func createTarGzWithSubdir(t *testing.T, dir string, fileName string, content string) string {
	t.Helper()
	tarGzPath := filepath.Join(dir, fileName)

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// 子目录中的 opsxcli 文件
	hdr := &tar.Header{
		Name: "subdir/opsxcli",
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))
	return tarGzPath
}

func TestExtractTarGz_正常解压(t *testing.T) {
	// 测试正常解压包含 opsxcli 的 tar.gz 文件
	tmpDir := t.TempDir()
	content := "#!/bin/bash\necho hello"
	tarGzPath := createTarGz(t, tmpDir, "test.tar.gz", content)

	destDir := t.TempDir()
	err := extractTarGz(tarGzPath, destDir)
	require.NoError(t, err)

	// 验证解压后的文件存在且内容正确
	extractedPath := filepath.Join(destDir, "opsxcli")
	data, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestExtractTarGz_子目录中的opsxcli(t *testing.T) {
	// 测试解压子目录中的 opsxcli 文件（HasSuffix 匹配）
	tmpDir := t.TempDir()
	content := "binary content"
	tarGzPath := createTarGzWithSubdir(t, tmpDir, "test.tar.gz", content)

	destDir := t.TempDir()
	err := extractTarGz(tarGzPath, destDir)
	require.NoError(t, err)

	extractedPath := filepath.Join(destDir, "opsxcli")
	data, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestExtractTarGz_忽略非opsxcli文件(t *testing.T) {
	// 测试 tar.gz 中只包含非 opsxcli 文件时应报错
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// 添加一个非 opsxcli 文件
	content := "hello world"
	hdr := &tar.Header{
		Name: "readme.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := io.WriteString(tw, content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err = extractTarGz(tarGzPath, destDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到 opsxcli 二进制文件")

	// readme.txt 不应被解压
	_, err = os.Stat(filepath.Join(destDir, "readme.txt"))
	assert.True(t, os.IsNotExist(err))
}

func TestExtractTarGz_文件不存在(t *testing.T) {
	// 测试解压不存在的文件
	err := extractTarGz("/nonexistent/path/file.tar.gz", t.TempDir())
	assert.Error(t, err)
}

func TestExtractTarGz_无效gzip文件(t *testing.T) {
	// 测试解压无效的 gzip 文件
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "invalid.tar.gz")
	require.NoError(t, os.WriteFile(tarGzPath, []byte("not a gzip file"), 0644))

	err := extractTarGz(tarGzPath, t.TempDir())
	assert.Error(t, err)
}

// --- copyFile ---

func TestCopyFile_正常复制(t *testing.T) {
	// 测试正常文件复制
	tmpDir := t.TempDir()
	content := "测试文件内容\n第二行"
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	require.NoError(t, os.WriteFile(src, []byte(content), 0644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestCopyFile_保留文件权限(t *testing.T) {
	// 测试复制时保留文件权限
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.sh")
	dst := filepath.Join(tmpDir, "dest.sh")

	require.NoError(t, os.WriteFile(src, []byte("#!/bin/bash"), 0755))

	err := copyFile(src, dst)
	require.NoError(t, err)

	srcInfo, _ := os.Stat(src)
	dstInfo, _ := os.Stat(dst)
	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
}

func TestCopyFile_源文件不存在(t *testing.T) {
	// 测试源文件不存在
	err := copyFile("/nonexistent/file.txt", filepath.Join(t.TempDir(), "dest.txt"))
	assert.Error(t, err)
}

func TestCopyFile_目标目录不存在(t *testing.T) {
	// 测试目标目录不存在
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	require.NoError(t, os.WriteFile(src, []byte("content"), 0644))

	dst := filepath.Join(tmpDir, "nonexistent", "dest.txt")
	err := copyFile(src, dst)
	assert.Error(t, err)
}

func TestCopyFile_覆盖已存在文件(t *testing.T) {
	// 测试覆盖已存在的目标文件
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	require.NoError(t, os.WriteFile(src, []byte("新内容"), 0644))
	require.NoError(t, os.WriteFile(dst, []byte("旧内容"), 0644))

	err := copyFile(src, dst)
	require.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "新内容", string(data))
}

// --- replaceFile ---

func TestReplaceFile_正常替换(t *testing.T) {
	// 测试正常替换文件（通过临时文件+rename）
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "new_binary")
	dst := filepath.Join(tmpDir, "old_binary")

	require.NoError(t, os.WriteFile(src, []byte("新版本内容"), 0755))
	require.NoError(t, os.WriteFile(dst, []byte("旧版本内容"), 0644))

	err := replaceFile(src, dst)
	require.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "新版本内容", string(data))

	// 验证权限被设置为源文件的权限
	dstInfo, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), dstInfo.Mode())
}

func TestReplaceFile_源文件不存在(t *testing.T) {
	// 测试源文件不存在
	err := replaceFile("/nonexistent/src", "/nonexistent/dst")
	assert.Error(t, err)
}

func TestReplaceFile_目标文件不存在(t *testing.T) {
	// 测试目标文件不存在（os.Stat 失败）
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	require.NoError(t, os.WriteFile(src, []byte("内容"), 0644))

	dst := filepath.Join(tmpDir, "nonexistent", "dest.txt")
	err := replaceFile(src, dst)
	assert.Error(t, err)
}

func TestReplaceFile_大文件替换(t *testing.T) {
	// 测试替换较大的文件内容
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "new_large")
	dst := filepath.Join(tmpDir, "old_large")

	// 创建大文件（1MB）
	largeContent := bytes.Repeat([]byte("A"), 1024*1024)
	require.NoError(t, os.WriteFile(src, largeContent, 0755))
	require.NoError(t, os.WriteFile(dst, []byte("old"), 0644))

	err := replaceFile(src, dst)
	require.NoError(t, err)

	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, len(largeContent), len(data))
}

// --- cleanOldBackups ---

func TestCleanOldBackups_无需清理(t *testing.T) {
	// 备份数量不超过 maxBackups，无需清理
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "opsxcli")
	require.NoError(t, os.WriteFile(binaryPath, []byte("binary"), 0755))

	// 创建少于 maxBackups 的备份（使用唯一文件名避免时间戳冲突）
	for i := 0; i < maxBackups; i++ {
		backupPath := filepath.Join(tmpDir, fmt.Sprintf("opsxcli.backup.2024010100000%d", i))
		require.NoError(t, os.WriteFile(backupPath, []byte("backup"), 0644))
	}

	err := cleanOldBackups(binaryPath)
	assert.NoError(t, err)

	// 验证所有备份仍存在
	pattern := filepath.Join(tmpDir, "opsxcli.backup.*")
	backups, _ := filepath.Glob(pattern)
	assert.Equal(t, maxBackups, len(backups))
}

func TestCleanOldBackups_清理多余备份(t *testing.T) {
	// 备份数量超过 maxBackups，应清理旧的
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "opsxcli")
	require.NoError(t, os.WriteFile(binaryPath, []byte("binary"), 0755))

	// 创建超过 maxBackups 的备份（4 个，应保留 2 个）
	var backupPaths []string
	for i := 0; i < maxBackups+2; i++ {
		backupPath := filepath.Join(tmpDir, fmt.Sprintf("opsxcli.backup.20240101%06d", i))
		require.NoError(t, os.WriteFile(backupPath, []byte(fmt.Sprintf("backup%d", i)), 0644))
		backupPaths = append(backupPaths, backupPath)
		time.Sleep(20 * time.Millisecond) // 确保修改时间不同
	}

	err := cleanOldBackups(binaryPath)
	assert.NoError(t, err)

	// 验证只剩 maxBackups 个备份
	pattern := filepath.Join(tmpDir, "opsxcli.backup.*")
	backups, _ := filepath.Glob(pattern)
	assert.Equal(t, maxBackups, len(backups))
}

func TestCleanOldBackups_无备份文件(t *testing.T) {
	// 没有任何备份文件时不应报错
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "opsxcli")
	require.NoError(t, os.WriteFile(binaryPath, []byte("binary"), 0755))

	err := cleanOldBackups(binaryPath)
	assert.NoError(t, err)
}

func TestCleanOldBackups_恰好maxBackups个(t *testing.T) {
	// 恰好 maxBackups 个备份，不应清理任何文件
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "opsxcli")
	require.NoError(t, os.WriteFile(binaryPath, []byte("binary"), 0755))

	for i := 0; i < maxBackups; i++ {
		backupPath := filepath.Join(tmpDir, fmt.Sprintf("opsxcli.backup.2024010100000%d", i))
		require.NoError(t, os.WriteFile(backupPath, []byte("backup"), 0644))
	}

	err := cleanOldBackups(binaryPath)
	assert.NoError(t, err)

	pattern := filepath.Join(tmpDir, "opsxcli.backup.*")
	backups, _ := filepath.Glob(pattern)
	assert.Equal(t, maxBackups, len(backups))
}

// --- downloadFile 集成测试（writeCounter 覆盖）---

func TestDownloadFile_进度显示(t *testing.T) {
	// 测试下载带有 Content-Length 的文件，触发 printProgress
	content := bytes.Repeat([]byte("X"), 1000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "downloaded.txt")
	err := downloadFile(server.URL, dest)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestDownloadFile_空文件(t *testing.T) {
	// 测试下载空文件
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "empty.txt")
	err := downloadFile(server.URL, dest)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Empty(t, data)
}

// --- extractTarGz 综合测试 ---

func TestExtractTarGz_包含多个文件的tar(t *testing.T) {
	// 创建包含 opsxcli 和其他文件的 tar.gz
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "multi.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// 添加非 opsxcli 文件
	files := []struct {
		name    string
		content string
	}{
		{"readme.txt", "readme content"},
		{"opsxcli", "binary content"},
		{"config.yaml", "key: value"},
	}

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: 0755,
			Size: int64(len(f.content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := io.WriteString(tw, f.content)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err := extractTarGz(tarGzPath, destDir)
	require.NoError(t, err)

	// 只有 opsxcli 被解压
	data, err := os.ReadFile(filepath.Join(destDir, "opsxcli"))
	require.NoError(t, err)
	assert.Equal(t, "binary content", string(data))

	// 其他文件不应存在
	_, err = os.Stat(filepath.Join(destDir, "readme.txt"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(destDir, "config.yaml"))
	assert.True(t, os.IsNotExist(err))
}

func TestExtractTarGz_空tar(t *testing.T) {
	// 测试空的 tar.gz 文件（没有二进制文件）
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "empty.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err := extractTarGz(tarGzPath, destDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到 opsxcli 二进制文件")
}

// --- copyFile 与 replaceFile 联合测试 ---

// --- extractTarGz 带平台后缀文件名 ---

func TestExtractTarGz_带平台后缀的文件名(t *testing.T) {
	// 测试 tar 中文件名为 opsxcli-linux-amd64 的情况（v1.0.6 release 实际场景）
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := "#!/bin/bash\necho opsxcli-v1.0.6"
	hdr := &tar.Header{
		Name: "opsxcli-linux-amd64",
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := io.WriteString(tw, content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err = extractTarGz(tarGzPath, destDir)
	require.NoError(t, err)

	// 解压后的文件应重命名为 opsxcli
	extractedPath := filepath.Join(destDir, "opsxcli")
	data, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestExtractTarGz_带aarch64后缀的文件名(t *testing.T) {
	// 测试 tar 中文件名为 opsxcli-linux-aarch64 的情况
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := "arm64 binary content"
	hdr := &tar.Header{
		Name: "opsxcli-linux-aarch64",
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := io.WriteString(tw, content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err = extractTarGz(tarGzPath, destDir)
	require.NoError(t, err)

	extractedPath := filepath.Join(destDir, "opsxcli")
	data, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestExtractTarGz_多个opsxcli文件只解压第一个(t *testing.T) {
	// 测试 tar 中包含 opsxcli 和 opsxcli-linux-amd64 两个文件，只解压第一个
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	files := []struct {
		name    string
		content string
	}{
		{"opsxcli", "standard content"},
		{"opsxcli-linux-amd64", "platform-specific content"},
	}

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: 0755,
			Size: int64(len(f.content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := io.WriteString(tw, f.content)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err := extractTarGz(tarGzPath, destDir)
	require.NoError(t, err)

	// 应使用第一个找到的文件
	extractedPath := filepath.Join(destDir, "opsxcli")
	data, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, "standard content", string(data))
}

func TestExtractTarGz_完全无关的文件报错(t *testing.T) {
	// tar 中只有非 opsxcli 文件时应该报错
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := "random content"
	hdr := &tar.Header{
		Name: "random-binary",
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := io.WriteString(tw, content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err = extractTarGz(tarGzPath, destDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未找到 opsxcli 二进制文件")
}

func TestExtractTarGz_Windows格式(t *testing.T) {
	// 测试 Windows 的 opsxcli.exe 格式
	tmpDir := t.TempDir()
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := "windows binary"
	hdr := &tar.Header{
		Name: "opsxcli.exe",
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err := io.WriteString(tw, content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(tarGzPath, buf.Bytes(), 0644))

	destDir := t.TempDir()
	err = extractTarGz(tarGzPath, destDir)
	require.NoError(t, err)

	extractedPath := filepath.Join(destDir, "opsxcli")
	data, err := os.ReadFile(extractedPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestCopyFile后ReplaceFile(t *testing.T) {
	// 模拟升级流程：先备份，再替换
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "opsxcli")
	require.NoError(t, os.WriteFile(binaryPath, []byte("旧版本"), 0755))

	backupPath := binaryPath + ".backup.20240101120000"
	err := copyFile(binaryPath, backupPath)
	require.NoError(t, err)

	// 模拟新版本
	newBinary := filepath.Join(tmpDir, "new_opsxcli")
	require.NoError(t, os.WriteFile(newBinary, []byte("新版本"), 0755))

	err = replaceFile(newBinary, binaryPath)
	require.NoError(t, err)

	// 验证主文件已更新
	data, err := os.ReadFile(binaryPath)
	require.NoError(t, err)
	assert.Equal(t, "新版本", string(data))

	// 验证备份文件保持不变
	backupData, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "旧版本", string(backupData))
}
