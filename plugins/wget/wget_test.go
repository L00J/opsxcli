package wget

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === extractFilename 测试 ===

func TestExtractFilename(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"URL带文件名", "https://example.com/path/to/file.zip", "file.zip"},
		{"URL以斜杠结尾", "https://example.com/path/to/", "index.html"},
		{"URL带查询参数", "https://example.com/download.tar.gz?v=1.0&ts=123", "download.tar.gz"},
		{"简单文件名", "https://example.com/data.json", "data.json"},
		{"根路径", "https://example.com/", "index.html"},
		{"带端口的URL", "http://localhost:8080/app.tar", "app.tar"},
		{"URL带片段和查询参数", "https://example.com/file.txt?token=abc#section", "file.txt"},
		{"仅域名(无尾部斜杠)", "https://example.com", "example.com"},
		{"多层路径带文件名", "https://cdn.example.com/v2/releases/opsxcli-linux-amd64", "opsxcli-linux-amd64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFilename(tt.url)
			assert.Equal(t, tt.expected, got, "URL %q 的文件名应为 %q", tt.url, tt.expected)
		})
	}
}

// === copyWithProgress 测试 ===

func TestCopyWithProgress_Basic(t *testing.T) {
	data := "Hello, World!"
	src := strings.NewReader(data)
	var dst bytes.Buffer

	written, err := copyWithProgress(&dst, src, 0, int64(len(data)))
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), written)
	assert.Equal(t, data, dst.String())
}

func TestCopyWithProgress_Empty(t *testing.T) {
	src := strings.NewReader("")
	var dst bytes.Buffer

	written, err := copyWithProgress(&dst, src, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), written)
	assert.Equal(t, "", dst.String())
}

func TestCopyWithProgress_WithStartOffset(t *testing.T) {
	data := "Hello, World!"
	src := strings.NewReader(data)
	var dst bytes.Buffer

	written, err := copyWithProgress(&dst, src, 100, 100+int64(len(data)))
	require.NoError(t, err)
	assert.Equal(t, int64(100+int64(len(data))), written)
	assert.Equal(t, data, dst.String())
}

func TestCopyWithProgress_LargeData(t *testing.T) {
	data := strings.Repeat("A", 100*1024) // 100KB
	src := strings.NewReader(data)
	var dst bytes.Buffer

	written, err := copyWithProgress(&dst, src, 0, int64(len(data)))
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), written)
	assert.Equal(t, len(data), dst.Len())
}

func TestCopyWithProgress_WriteError(t *testing.T) {
	data := "Hello, World!"
	src := strings.NewReader(data)
	dst := &errorWriter{}

	_, err := copyWithProgress(dst, src, 0, int64(len(data)))
	assert.Error(t, err)
}

func TestCopyWithProgress_ShortWrite(t *testing.T) {
	data := "Hello, World!"
	src := strings.NewReader(data)
	dst := &shortWriter{}

	_, err := copyWithProgress(dst, src, 0, int64(len(data)))
	assert.Error(t, err)
}

func TestCopyWithProgress_ReadError(t *testing.T) {
	src := &errorReader{}
	var dst bytes.Buffer

	_, err := copyWithProgress(&dst, src, 0, 100)
	assert.Error(t, err)
}

func TestCopyWithProgress_ZeroTotal(t *testing.T) {
	data := "some data"
	src := strings.NewReader(data)
	var dst bytes.Buffer

	written, err := copyWithProgress(&dst, src, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(len(data)), written)
	assert.Equal(t, data, dst.String())
}

// === Download 集成测试（使用 httptest） ===

func TestDownload_BasicHTTP(t *testing.T) {
	content := "test file content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Wget/1.21.3", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "test.txt")

	err := Download(server.URL+"/file.txt", output, false)
	require.NoError(t, err)

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestDownload_ExtractFilenameWhenNoOutput(t *testing.T) {
	content := "hello"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	err := Download(server.URL+"/download.tar.gz", "", false)
	require.NoError(t, err)

	data, err := os.ReadFile("download.tar.gz")
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestDownload_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "test.txt")

	err := Download(server.URL+"/notfound", output, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDownload_ContinueDownload(t *testing.T) {
	partial := "full "

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte("file content"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("full file content"))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "partial.txt")

	err := os.WriteFile(output, []byte(partial), 0644)
	require.NoError(t, err)

	err = Download(server.URL+"/file.txt", output, true)
	require.NoError(t, err)

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, "full file content", string(data))
}

func TestDownload_ContinueNoPartialFile(t *testing.T) {
	content := "hello world"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "newfile.txt")

	err := Download(server.URL+"/file.txt", output, true)
	require.NoError(t, err)

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestDownload_InvalidURL(t *testing.T) {
	// 使用已经关闭的服务器，确保立即失败而不等待超时
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // 立即关闭

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "test.txt")

	err := Download(server.URL+"/file.txt", output, false)
	assert.Error(t, err)
}

func TestDownload_CannotCreateFile(t *testing.T) {
	content := "test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
	defer server.Close()

	err := Download(server.URL+"/file.txt", "/nonexistent/dir/file.txt", false)
	assert.Error(t, err)
}

func TestDownload_HTTPWithoutSSL(t *testing.T) {
	content := "http content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(content))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "http_test.txt")

	err := Download(server.URL+"/file.txt", output, false)
	require.NoError(t, err)

	data, err := os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

// === 测试辅助类型 ===

type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("write error")
}

type shortWriter struct{}

func (w *shortWriter) Write(p []byte) (n int, err error) {
	if len(p) > 1 {
		return 1, nil
	}
	return 0, nil
}

type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

var _ io.Reader = (*errorReader)(nil)
var _ io.Writer = (*errorWriter)(nil)
var _ io.Writer = (*shortWriter)(nil)
