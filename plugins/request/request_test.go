package request

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.bytes), func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestSendWithStatus_InvalidURL(t *testing.T) {
	err := SendWithStatus("://invalid", "GET", nil, "", "", false)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestSendWithStatus_BasicGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("X-Custom", "hello")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "response body")
	}))
	defer server.Close()

	// 不显示状态，避免 TUI 依赖
	err := SendWithStatus(server.URL, "GET", nil, "", "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() error: %v", err)
	}
}

func TestSendWithStatus_POST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected auto content-type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	err := SendWithStatus(server.URL, "POST", nil, "key=value", "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() POST error: %v", err)
	}
}

func TestSendWithStatus_CustomHeaders(t *testing.T) {
	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Test-Header")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := SendWithStatus(server.URL, "GET", []string{"X-Test-Header: my-value"}, "", "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() error: %v", err)
	}
	if receivedHeader != "my-value" {
		t.Errorf("X-Test-Header = %q, want %q", receivedHeader, "my-value")
	}
}

func TestSendWithStatus_CustomContentType(t *testing.T) {
	var receivedCT string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// 显式设置 Content-Type，不应被自动覆盖
	err := SendWithStatus(server.URL, "POST", []string{"Content-Type: application/json"}, `{"key":"val"}`, "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() error: %v", err)
	}
	if receivedCT != "application/json" {
		t.Errorf("Content-Type = %q, want %q", receivedCT, "application/json")
	}
}

func TestSendWithStatus_SaveOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "saved content")
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.txt")

	err := SendWithStatus(server.URL, "GET", nil, "", outputPath, false)
	if err != nil {
		t.Fatalf("SendWithStatus() save error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != "saved content" {
		t.Errorf("output file content = %q, want %q", string(data), "saved content")
	}
}

func TestSendWithStatus_InvalidHeader(t *testing.T) {
	// 没有冒号的 header 应被跳过
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := SendWithStatus(server.URL, "GET", []string{"invalid-header-no-colon"}, "", "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() with invalid header error: %v", err)
	}
}

func TestSend_BasicGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	err := Send(server.URL, "GET", nil, "", "")
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}
}

func TestSendWithStatus_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer server.Close()

	// 即使服务器返回 500，客户端也不应返回错误
	err := SendWithStatus(server.URL, "GET", nil, "", "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() should not error on 500, got: %v", err)
	}
}

func TestSendWithStatus_HeaderWithSpaces(t *testing.T) {
	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := SendWithStatus(server.URL, "GET", []string{"Authorization: Bearer token123"}, "", "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() error: %v", err)
	}
	if receivedHeader != "Bearer token123" {
		t.Errorf("Authorization = %q, want %q", receivedHeader, "Bearer token123")
	}
}

func TestSendWithStatus_PUT(t *testing.T) {
	var methodReceived string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := SendWithStatus(server.URL, "PUT", []string{"Content-Type: application/json"}, `{"update":true}`, "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() PUT error: %v", err)
	}
	if methodReceived != "PUT" {
		t.Errorf("method = %q, want PUT", methodReceived)
	}
}

func TestSendWithStatus_DELETE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := SendWithStatus(server.URL, "DELETE", nil, "", "", false)
	if err != nil {
		t.Fatalf("SendWithStatus() DELETE error: %v", err)
	}
}

func TestFormatBytes_LargeValues(t *testing.T) {
	tests := []struct {
		name  string
		bytes int
		want  string
	}{
		{"exact_KB", 1024, "1.0 KB"},
		{"exact_MB", 1048576, "1.0 MB"},
		{"1.5_MB", 1572864, "1.5 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if !strings.Contains(got, strings.Split(tt.want, " ")[1]) {
				t.Errorf("formatBytes(%d) = %q, want unit in %q", tt.bytes, got, tt.want)
			}
		})
	}
}
