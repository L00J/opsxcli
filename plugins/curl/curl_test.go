package curl

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequest_AutoHTTPS(t *testing.T) {
	var requestedHost string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedHost = r.Host
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	// 提取 host:port 从 server.URL
	hostPort := strings.TrimPrefix(server.URL, "http://")

	// 使用 http:// 前缀，因为 httptest 默认是 http 服务器
	err := Request("http://"+hostPort+"/test", "GET", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
	if !strings.Contains(requestedHost, hostPort) {
		t.Errorf("host = %q, want to contain %q", requestedHost, hostPort)
	}
}

func TestRequest_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("User-Agent") != "curl/8.0.1" {
			t.Errorf("expected curl User-Agent, got %s", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "hello world")
	}))
	defer server.Close()

	err := Request(server.URL, "GET", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() GET error: %v", err)
	}
}

func TestRequest_POSTAutoSwitch(t *testing.T) {
	// 有数据但 method 为 GET，应自动改为 POST
	var methodReceived string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	err := Request(server.URL, "GET", nil, "data=test", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
	if methodReceived != "POST" {
		t.Errorf("method = %q, want POST (auto-switched from GET with data)", methodReceived)
	}
}

func TestRequest_CustomHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Request(server.URL, "GET", []string{"Authorization: Bearer test-token"}, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
	if receivedAuth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer test-token")
	}
}

func TestRequest_ContentTypeAuto(t *testing.T) {
	var ct string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Request(server.URL, "POST", nil, "key=value", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want auto-set form-urlencoded", ct)
	}
}

func TestRequest_CustomContentType(t *testing.T) {
	var ct string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Request(server.URL, "POST", []string{"Content-Type: application/json"}, `{"k":"v"}`, "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestRequest_SaveOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "file content here")
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.txt")

	err := Request(server.URL, "GET", nil, "", outputPath, false, false, true)
	if err != nil {
		t.Fatalf("Request() save error: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(data) != "file content here" {
		t.Errorf("output = %q, want %q", string(data), "file content here")
	}
}

func TestRequest_NoRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			w.Header().Set("Location", "/final")
			w.WriteHeader(http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "final destination")
	}))
	defer server.Close()

	// 不跟随重定向
	err := Request(server.URL+"/redirect", "GET", nil, "", "", false, false, false)
	if err != nil {
		t.Fatalf("Request() no-redirect error: %v", err)
	}
}

func TestRequest_HEAD(t *testing.T) {
	var methodReceived string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Request(server.URL, "HEAD", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() HEAD error: %v", err)
	}
	if methodReceived != "HEAD" {
		t.Errorf("method = %q, want HEAD", methodReceived)
	}
}

func TestRequest_InvalidURL(t *testing.T) {
	err := Request("://invalid-host", "GET", nil, "", "", false, false, true)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestRequest_DELETE(t *testing.T) {
	var methodReceived string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methodReceived = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := Request(server.URL, "DELETE", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() DELETE error: %v", err)
	}
	if methodReceived != "DELETE" {
		t.Errorf("method = %q, want DELETE", methodReceived)
	}
}

func TestRequest_InvalidHeaderSkipped(t *testing.T) {
	// 没有 colon 的 header 应被静默跳过
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Request(server.URL, "GET", []string{"no-colon-here"}, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() with bad header error: %v", err)
	}
}
