package curl

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequest_Verbose 测试 verbose 模式输出请求/响应详情
func TestRequest_Verbose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "verbose response body")
	}))
	defer server.Close()

	err := Request(server.URL+"/path", "GET", nil, "", "", false, true, true)
	if err != nil {
		t.Fatalf("Request() verbose error: %v", err)
	}
}

// TestRequest_IncludeHeaders 测试 -i 参数输出响应头
func TestRequest_IncludeHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "body content")
	}))
	defer server.Close()

	err := Request(server.URL, "GET", nil, "", "", true, false, true)
	if err != nil {
		t.Fatalf("Request() includeHeaders error: %v", err)
	}
}

// TestRequest_VerboseWithPost 测试 verbose + POST 组合
func TestRequest_VerboseWithPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, "created")
	}))
	defer server.Close()

	err := Request(server.URL, "POST", nil, "key=value", "", false, true, true)
	if err != nil {
		t.Fatalf("Request() verbose POST error: %v", err)
	}
}

// TestRequest_BodyWithNewline 测试响应体带换行符
func TestRequest_BodyWithNewline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "line1\nline2\n")
	}))
	defer server.Close()

	err := Request(server.URL, "GET", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
}

// TestRequest_BodyWithoutNewline 测试响应体不带换行符（自动添加换行）
func TestRequest_BodyWithoutNewline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "no trailing newline")
	}))
	defer server.Close()

	err := Request(server.URL, "GET", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
}

// TestRequest_EmptyBody 测试空响应体
func TestRequest_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := Request(server.URL, "GET", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
}

// TestRequest_AutoPrefixNoHTTP 测试 URL 自动添加 https://
func TestRequest_AutoPrefixNoHTTP(t *testing.T) {
	var requested bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	// 用 http:// 因为 httptest 是 http 服务器
	hostPort := strings.TrimPrefix(server.URL, "http://")
	err := Request("http://"+hostPort, "GET", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() auto-prefix error: %v", err)
	}
	if !requested {
		t.Error("expected request to be made")
	}
}

// TestRequest_VerboseAndIncludeHeaders 测试 verbose + includeHeaders（verbose 优先）
func TestRequest_VerboseAndIncludeHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer server.Close()

	err := Request(server.URL, "GET", nil, "", "", true, true, true)
	if err != nil {
		t.Fatalf("Request() error: %v", err)
	}
}

// TestRequest_HEADNoBodyOutput 测试 HEAD 请求不输出 body
func TestRequest_HEADNoBodyOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "should not appear in output")
	}))
	defer server.Close()

	err := Request(server.URL, "HEAD", nil, "", "", false, false, true)
	if err != nil {
		t.Fatalf("Request() HEAD error: %v", err)
	}
}
