package cloudhost

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- HostEntry ---

func TestHostEntry(t *testing.T) {
	entry := HostEntry{Hostname: "web01", IP: "192.168.1.10"}
	assert.Equal(t, "web01", entry.Hostname)
	assert.Equal(t, "192.168.1.10", entry.IP)
}

// --- ConsulServiceRegistration JSON序列化 ---

func TestConsulServiceRegistration(t *testing.T) {
	reg := ConsulServiceRegistration{
		ID:      "web01-app",
		Name:    "webapp",
		Address: "192.168.1.10",
		Port:    8080,
		Checks: []ConsulServiceCheck{
			{HTTP: "http://192.168.1.10:8080/health", Interval: "10s"},
		},
	}
	assert.Equal(t, "web01-app", reg.ID)
	assert.Equal(t, "webapp", reg.Name)
	assert.Equal(t, "http://192.168.1.10:8080/health", reg.Checks[0].HTTP)

	// 验证JSON序列化
	data, err := json.Marshal(reg)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"id":"web01-app"`)
	assert.Contains(t, string(data), `"name":"webapp"`)
	assert.Contains(t, string(data), `"port":8080`)
}

// --- ConsulHealthCheckResult JSON反序列化 ---

func TestConsulHealthCheckResult(t *testing.T) {
	result := ConsulHealthCheckResult{ServiceID: "svc-001"}
	assert.Equal(t, "svc-001", result.ServiceID)

	// 验证JSON反序列化
	jsonStr := `{"ServiceID":"web01-node-exporter"}`
	var parsed ConsulHealthCheckResult
	err := json.Unmarshal([]byte(jsonStr), &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "web01-node-exporter", parsed.ServiceID)
}

// --- ConsulHealthCheckResult 空ServiceID ---

func TestConsulHealthCheckResult_EmptyServiceID(t *testing.T) {
	result := ConsulHealthCheckResult{}
	assert.Equal(t, "", result.ServiceID)
}

// --- NewCloudHostRegistry ---

func TestNewCloudHostRegistry(t *testing.T) {
	hosts := []HostEntry{
		{Hostname: "host1", IP: "10.0.0.1"},
		{Hostname: "host2", IP: "10.0.0.2"},
	}
	registry := NewCloudHostRegistry("http://consul:8500/", hosts, 8080, 9100, "/metrics")
	assert.NotNil(t, registry)
	assert.Equal(t, "http://consul:8500", registry.consulURL) // trailing slash trimmed
	assert.Equal(t, 8080, registry.appPort)
	assert.Equal(t, 9100, registry.nodeExpPort)
	assert.Equal(t, "/metrics", registry.metricsPath)
	assert.Len(t, registry.hosts, 2)
}

func TestNewCloudHostRegistry_EmptyHosts(t *testing.T) {
	registry := NewCloudHostRegistry("http://localhost:8500", nil, 80, 9100, "/metrics")
	assert.NotNil(t, registry)
	assert.Empty(t, registry.hosts)
}

func TestNewCloudHostRegistry_TrailingSlashTrimmed(t *testing.T) {
	registry := NewCloudHostRegistry("http://consul:8500/", nil, 80, 9100, "/metrics")
	assert.NotNil(t, registry)
	assert.Equal(t, "http://consul:8500", registry.consulURL)
}

func TestNewCloudHostRegistry_MultipleTrailingSlashes(t *testing.T) {
	// TrimSuffix 只去掉一个 /，多余的保留
	registry := NewCloudHostRegistry("http://consul:8500///", nil, 80, 9100, "/metrics")
	assert.NotNil(t, registry)
	assert.Equal(t, "http://consul:8500//", registry.consulURL)
}

func TestNewCloudHostRegistry_HttpClient(t *testing.T) {
	registry := NewCloudHostRegistry("http://consul:8500", nil, 80, 9100, "/metrics")
	assert.NotNil(t, registry.httpClient)
}

// --- ParseHosts ---

func TestParseHosts_ColonFormat(t *testing.T) {
	result := ParseHosts([]string{"web01:192.168.1.10", "web02:192.168.1.20"})
	assert.Len(t, result, 2)
	assert.Equal(t, HostEntry{Hostname: "web01", IP: "192.168.1.10"}, result[0])
	assert.Equal(t, HostEntry{Hostname: "web02", IP: "192.168.1.20"}, result[1])
}

func TestParseHosts_CommaFormat(t *testing.T) {
	result := ParseHosts([]string{"web01,192.168.1.10", "web02,192.168.1.20"})
	assert.Len(t, result, 2)
	assert.Equal(t, HostEntry{Hostname: "web01", IP: "192.168.1.10"}, result[0])
	assert.Equal(t, HostEntry{Hostname: "web02", IP: "192.168.1.20"}, result[1])
}

func TestParseHosts_MixedFormats(t *testing.T) {
	result := ParseHosts([]string{"web01:192.168.1.10", "web02,192.168.1.20"})
	assert.Len(t, result, 2)
	assert.Equal(t, "web01", result[0].Hostname)
	assert.Equal(t, "web02", result[1].Hostname)
}

func TestParseHosts_InvalidFormat(t *testing.T) {
	result := ParseHosts([]string{"invalidhost"})
	assert.Empty(t, result) // 无效格式跳过
}

func TestParseHosts_Empty(t *testing.T) {
	result := ParseHosts([]string{})
	assert.Empty(t, result)
}

func TestParseHosts_Nil(t *testing.T) {
	result := ParseHosts(nil)
	assert.Empty(t, result)
}

func TestParseHosts_WhitespaceTrimmed(t *testing.T) {
	result := ParseHosts([]string{" web01 : 192.168.1.10 "})
	assert.Len(t, result, 1)
	assert.Equal(t, "web01", result[0].Hostname)
	assert.Equal(t, "192.168.1.10", result[0].IP)
}

func TestParseHosts_ColonInIP(t *testing.T) {
	// IPv6 地址包含冒号，SplitN 限制为2所以第一个冒号分割
	result := ParseHosts([]string{"web01:::1"})
	assert.Len(t, result, 1)
	assert.Equal(t, "web01", result[0].Hostname)
	assert.Equal(t, "::1", result[0].IP) // SplitN(2) 在第一个冒号分割
}

// --- LoadHostsFromFile ---

func TestLoadHostsFromFile_ColonFormat(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/hosts.txt"
	content := "web01:192.168.1.10\nweb02:192.168.1.20\n"
	err := os.WriteFile(file, []byte(content), 0644)
	assert.NoError(t, err)

	hosts, err := LoadHostsFromFile(file)
	assert.NoError(t, err)
	assert.Len(t, hosts, 2)
	assert.Equal(t, HostEntry{Hostname: "web01", IP: "192.168.1.10"}, hosts[0])
	assert.Equal(t, HostEntry{Hostname: "web02", IP: "192.168.1.20"}, hosts[1])
}

func TestLoadHostsFromFile_CommaFormat(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/hosts.txt"
	content := "web01,192.168.1.10\nweb02,192.168.1.20\n"
	err := os.WriteFile(file, []byte(content), 0644)
	assert.NoError(t, err)

	hosts, err := LoadHostsFromFile(file)
	assert.NoError(t, err)
	assert.Len(t, hosts, 2)
}

func TestLoadHostsFromFile_Comments(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/hosts.txt"
	content := "# 这是一个注释\nweb01:192.168.1.10\n# 另一个注释\nweb02:192.168.1.20\n"
	err := os.WriteFile(file, []byte(content), 0644)
	assert.NoError(t, err)

	hosts, err := LoadHostsFromFile(file)
	assert.NoError(t, err)
	assert.Len(t, hosts, 2)
}

func TestLoadHostsFromFile_EmptyLines(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/hosts.txt"
	content := "\nweb01:192.168.1.10\n\n\nweb02:192.168.1.20\n\n"
	err := os.WriteFile(file, []byte(content), 0644)
	assert.NoError(t, err)

	hosts, err := LoadHostsFromFile(file)
	assert.NoError(t, err)
	assert.Len(t, hosts, 2)
}

func TestLoadHostsFromFile_MixedFormats(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/hosts.txt"
	content := "web01:192.168.1.10\nweb02,192.168.1.20\ninvalidline\n"
	err := os.WriteFile(file, []byte(content), 0644)
	assert.NoError(t, err)

	hosts, err := LoadHostsFromFile(file)
	assert.NoError(t, err)
	assert.Len(t, hosts, 2)
}

func TestLoadHostsFromFile_FileNotFound(t *testing.T) {
	hosts, err := LoadHostsFromFile("/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Nil(t, hosts)
}

func TestLoadHostsFromFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/hosts.txt"
	err := os.WriteFile(file, []byte(""), 0644)
	assert.NoError(t, err)

	hosts, err := LoadHostsFromFile(file)
	assert.NoError(t, err)
	assert.Empty(t, hosts)
}

func TestLoadHostsFromFile_CommentsOnly(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/hosts.txt"
	content := "# comment 1\n# comment 2\n"
	err := os.WriteFile(file, []byte(content), 0644)
	assert.NoError(t, err)

	hosts, err := LoadHostsFromFile(file)
	assert.NoError(t, err)
	assert.Empty(t, hosts)
}

// --- checkEndpoint (用 httptest mock) ---

func TestCheckEndpoint_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "system_cpu_usage 0.5")
	}))
	defer server.Close()

	registry := NewCloudHostRegistry("http://consul:8500", nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	// 使用测试服务器的URL
	result := registry.checkEndpoint(server.URL+"/metrics", "system_cpu_usage")
	assert.True(t, result)
}

func TestCheckEndpoint_NoExpectedText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "some content")
	}))
	defer server.Close()

	registry := NewCloudHostRegistry("http://consul:8500", nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	// 不验证内容，只验证状态码
	result := registry.checkEndpoint(server.URL+"/metrics", "")
	assert.True(t, result)
}

func TestCheckEndpoint_ExpectedTextNotMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "other_content")
	}))
	defer server.Close()

	registry := NewCloudHostRegistry("http://consul:8500", nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	result := registry.checkEndpoint(server.URL+"/metrics", "system_cpu_usage")
	assert.False(t, result)
}

func TestCheckEndpoint_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	registry := NewCloudHostRegistry("http://consul:8500", nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	result := registry.checkEndpoint(server.URL+"/metrics", "")
	assert.False(t, result)
}

func TestCheckEndpoint_ConnectionFailed(t *testing.T) {
	registry := NewCloudHostRegistry("http://consul:8500", nil, 8080, 9100, "/metrics")
	// 使用一个不可达的地址
	result := registry.checkEndpoint("http://127.0.0.1:1/metrics", "")
	assert.False(t, result)
}

func TestCheckEndpoint_InvalidURL(t *testing.T) {
	registry := NewCloudHostRegistry("http://consul:8500", nil, 8080, 9100, "/metrics")
	result := registry.checkEndpoint("://invalid-url", "")
	assert.False(t, result)
}

// --- registerService (用 httptest mock Consul API) ---

func TestRegisterService_Success(t *testing.T) {
	var receivedBody ConsulServiceRegistration
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/v1/agent/service/register", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	registry := NewCloudHostRegistry(server.URL, nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	err := registry.registerService("web01-app", "application", "192.168.1.10", 8080, "http://192.168.1.10:8080/metrics")
	assert.NoError(t, err)
	assert.Equal(t, "web01-app", receivedBody.ID)
	assert.Equal(t, "application", receivedBody.Name)
	assert.Equal(t, "192.168.1.10", receivedBody.Address)
	assert.Equal(t, 8080, receivedBody.Port)
	assert.Equal(t, "http://192.168.1.10:8080/metrics", receivedBody.Checks[0].HTTP)
	assert.Equal(t, "5s", receivedBody.Checks[0].Interval)
}

func TestRegisterService_ConsulError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer server.Close()

	registry := NewCloudHostRegistry(server.URL, nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	err := registry.registerService("web01-app", "application", "192.168.1.10", 8080, "http://192.168.1.10:8080/metrics")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestRegisterService_ConnectionFailed(t *testing.T) {
	registry := NewCloudHostRegistry("http://127.0.0.1:1", nil, 8080, 9100, "/metrics")
	err := registry.registerService("web01-app", "application", "192.168.1.10", 8080, "http://192.168.1.10:8080/metrics")
	assert.Error(t, err)
}

// --- CleanFailedInstances (用 httptest mock) ---

func TestCleanFailedInstances_NoCritical(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/health/state/critical", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "[]")
	}))
	defer server.Close()

	registry := NewCloudHostRegistry(server.URL, nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	err := registry.CleanFailedInstances()
	assert.NoError(t, err)
}

func TestCleanFailedInstances_WithCritical(t *testing.T) {
	deregistered := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health/state/critical" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `[{"ServiceID":"svc-001"},{"ServiceID":"svc-002"},{"ServiceID":""}]`)
		} else if strings.HasPrefix(r.URL.Path, "/v1/agent/service/deregister/") {
			serviceID := strings.TrimPrefix(r.URL.Path, "/v1/agent/service/deregister/")
			deregistered = append(deregistered, serviceID)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	registry := NewCloudHostRegistry(server.URL, nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	err := registry.CleanFailedInstances()
	assert.NoError(t, err)
	assert.Len(t, deregistered, 2)
	assert.Contains(t, deregistered, "svc-001")
	assert.Contains(t, deregistered, "svc-002")
}

func TestCleanFailedInstances_ConsulError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	registry := NewCloudHostRegistry(server.URL, nil, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	err := registry.CleanFailedInstances()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestCleanFailedInstances_ConnectionFailed(t *testing.T) {
	registry := NewCloudHostRegistry("http://127.0.0.1:1", nil, 8080, 9100, "/metrics")
	err := registry.CleanFailedInstances()
	assert.Error(t, err)
}

// --- RegisterAppServices (用 httptest mock) ---

func TestRegisterAppServices_SkipUnavailable(t *testing.T) {
	// 模拟 metrics 端点不可用
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 所有请求返回 404
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	hosts := []HostEntry{{Hostname: "host1", IP: "127.0.0.1"}}
	registry := NewCloudHostRegistry(server.URL, hosts, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	// 不会 panic，只打印日志
	registry.RegisterAppServices()
}

func TestRegisterAppServices_AvailableAndRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/metrics") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "system_cpu_usage 0.5")
		} else if r.URL.Path == "/v1/agent/service/register" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	hosts := []HostEntry{{Hostname: "host1", IP: "127.0.0.1"}}
	registry := NewCloudHostRegistry(server.URL, hosts, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	// 注意：checkEndpoint URL 使用 host.IP + appPort 而非 server URL，
	// 所以实际不会 hit 到 mock server。此测试确保不会 panic。
	registry.RegisterAppServices()
}

// --- RegisterNodeExporters (用 httptest mock) ---

func TestRegisterNodeExporters_Unavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	hosts := []HostEntry{{Hostname: "host1", IP: "127.0.0.1"}}
	registry := NewCloudHostRegistry(server.URL, hosts, 8080, 9100, "/metrics")
	registry.httpClient = server.Client()

	// 不会 panic
	registry.RegisterNodeExporters()
}
