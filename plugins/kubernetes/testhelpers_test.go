package kubernetes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// newMockClient 创建一个指向 mock 服务器的 K8sHTTPClient
func newMockClient(t *testing.T, handler http.Handler) *K8sHTTPClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &K8sHTTPClient{
		baseURL:    server.URL,
		httpClient: server.Client(),
		token:      "test-token",
	}
}

// newMockClientWithMux 创建一个指向 mock 服务器的 K8sHTTPClient，使用自定义 mux
func newMockClientWithMux(t *testing.T, setup func(mux *http.ServeMux)) *K8sHTTPClient {
	t.Helper()
	mux := http.NewServeMux()
	setup(mux)
	return newMockClient(t, mux)
}

// jsonResponse 写入 JSON 响应
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// mustMarshal 将对象序列化为 JSON，失败则 panic
func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// verifyAuth 验证请求中的 Authorization 头
func verifyAuth(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer test-token"
}

// newMockMonitor 创建一个用于测试的 KubernetesMonitor
func newMockMonitor(t *testing.T, handler http.Handler) *KubernetesMonitor {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &KubernetesMonitor{
		consulURL:   server.URL,
		metricsPath: "/metrics",
		cacheDir:    t.TempDir(),
		skipCheck:   true, // 跳过 Prometheus 端点检查
		client: &K8sHTTPClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
			token:      "test-token",
		},
		httpClient: server.Client(),
	}
}

// newMockMonitorWithMux 创建一个带自定义 mux 的 KubernetesMonitor
func newMockMonitorWithMux(t *testing.T, setup func(mux *http.ServeMux)) *KubernetesMonitor {
	t.Helper()
	mux := http.NewServeMux()
	setup(mux)
	return newMockMonitor(t, mux)
}

// setupK8sAPI 设置常见的 K8s API mock 路由
func setupK8sAPI(mux *http.ServeMux) {
	// Nodes
	mux.HandleFunc("/api/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})

	// Namespaces
	mux.HandleFunc("/api/v1/namespaces", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})

	// Services (all namespaces)
	mux.HandleFunc("/api/v1/services", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})

	// Deployments (all namespaces)
	mux.HandleFunc("/apis/apps/v1/deployments", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})

	// Pods (all namespaces)
	mux.HandleFunc("/api/v1/pods", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})

	// Ingresses (all namespaces)
	mux.HandleFunc("/apis/networking.k8s.io/v1/ingresses", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})

	// Metrics
	mux.HandleFunc("/apis/metrics.k8s.io/v1beta1/pods", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})

	mux.HandleFunc("/apis/metrics.k8s.io/v1beta1/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !verifyAuth(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"items": []interface{}{},
		})
	})
}

// requireNoError 断言没有错误
func requireNoError(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err)
}
