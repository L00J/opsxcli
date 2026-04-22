package docker

import (
	"encoding/json"
	"testing"
	"time"
)

// TestRegistrySourcesResponse_Parse 测试 JSON 反序列化
func TestRegistrySourcesResponse_Parse(t *testing.T) {
	raw := `{
		"success": true,
		"message": "ok",
		"registries": ["docker.aityp.com", "docker.1ms.run", "mirror.example.com"],
		"updated_at": "2025-12-19T15:00:00Z"
	}`

	var resp RegistrySourcesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}

	if !resp.Success {
		t.Fatal("Success 应为 true")
	}
	if resp.Message != "ok" {
		t.Fatalf("Message = %q，期望 %q", resp.Message, "ok")
	}
	if len(resp.Registries) != 3 {
		t.Fatalf("Registries 数量 = %d，期望 3", len(resp.Registries))
	}
	if resp.Registries[0] != "docker.aityp.com" {
		t.Fatalf("Registries[0] = %q", resp.Registries[0])
	}
	if resp.UpdatedAt != "2025-12-19T15:00:00Z" {
		t.Fatalf("UpdatedAt = %q", resp.UpdatedAt)
	}
}

// TestRegistrySourcesResponse_Minimal 最小化 JSON（省略 optional 字段）
func TestRegistrySourcesResponse_Minimal(t *testing.T) {
	raw := `{"success": false, "registries": []}`

	var resp RegistrySourcesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}
	if resp.Success {
		t.Fatal("Success 应为 false")
	}
	if resp.Message != "" {
		t.Fatalf("Message 应为空，实际 %q", resp.Message)
	}
	if len(resp.Registries) != 0 {
		t.Fatalf("Registries 应为空切片")
	}
	if resp.UpdatedAt != "" {
		t.Fatalf("UpdatedAt 应为空，实际 %q", resp.UpdatedAt)
	}
}

// TestPrivateRegistryAuth_Serialize 测试 JSON 序列化
func TestPrivateRegistryAuth_Serialize(t *testing.T) {
	auth := PrivateRegistryAuth{
		Registry: "private.example.com",
		Username: "user@example.com",
		Password: "secret123",
		Email:    "user@example.com",
	}

	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 验证序列化后能还原
	var auth2 PrivateRegistryAuth
	if err := json.Unmarshal(data, &auth2); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}
	if auth2.Registry != auth.Registry {
		t.Fatalf("Registry = %q，期望 %q", auth2.Registry, auth.Registry)
	}
	if auth2.Username != auth.Username {
		t.Fatalf("Username = %q，期望 %q", auth2.Username, auth.Username)
	}
	if auth2.Password != auth.Password {
		t.Fatalf("Password = %q，期望 %q", auth2.Password, auth.Password)
	}
	if auth2.Email != auth.Email {
		t.Fatalf("Email = %q，期望 %q", auth2.Email, auth.Email)
	}
}

// TestPrivateRegistryAuth_OmitEmptyEmail Email 为空时 omitempty
func TestPrivateRegistryAuth_OmitEmptyEmail(t *testing.T) {
	auth := PrivateRegistryAuth{
		Registry: "private.example.com",
		Username: "user",
		Password: "pass",
		// Email 为空
	}

	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// email 字段不应出现
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("解析 JSON map 失败: %v", err)
	}
	if _, ok := m["email"]; ok {
		t.Fatal("email 字段应为 omitempty，不应出现在 JSON 中")
	}
}

// TestNewRegistrySourcesAPI 构造验证
func TestNewRegistrySourcesAPI(t *testing.T) {
	api := NewRegistrySourcesAPI("https://api.example.com/v1/docker/registries", "test-token")
	if api == nil {
		t.Fatal("NewRegistrySourcesAPI 返回 nil")
	}
	if api.Endpoint != "https://api.example.com/v1/docker/registries" {
		t.Fatalf("Endpoint = %q", api.Endpoint)
	}
	if api.Token != "test-token" {
		t.Fatalf("Token = %q", api.Token)
	}
	if api.Timeout != 10*time.Second {
		t.Fatalf("Timeout = %v，期望 10s", api.Timeout)
	}
}

// TestNewRegistrySourcesAPI_EmptyToken 空 token 也可构造
func TestNewRegistrySourcesAPI_EmptyToken(t *testing.T) {
	api := NewRegistrySourcesAPI("https://api.example.com", "")
	if api.Token != "" {
		t.Fatalf("Token 应为空，实际 %q", api.Token)
	}
}

// TestRegistrySourcesResponse_RoundTrip 序列化→反序列化 往返一致性
func TestRegistrySourcesResponse_RoundTrip(t *testing.T) {
	original := RegistrySourcesResponse{
		Success:    true,
		Message:    "测试消息",
		Registries: []string{"a.com", "b.com"},
		UpdatedAt:  "2026-01-01T00:00:00Z",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var restored RegistrySourcesResponse
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if restored.Success != original.Success {
		t.Fatalf("Success: %v ≠ %v", restored.Success, original.Success)
	}
	if restored.Message != original.Message {
		t.Fatalf("Message: %q ≠ %q", restored.Message, original.Message)
	}
	if len(restored.Registries) != len(original.Registries) {
		t.Fatalf("Registries 长度: %d ≠ %d", len(restored.Registries), len(original.Registries))
	}
	for i, r := range restored.Registries {
		if r != original.Registries[i] {
			t.Fatalf("Registries[%d]: %q ≠ %q", i, r, original.Registries[i])
		}
	}
	if restored.UpdatedAt != original.UpdatedAt {
		t.Fatalf("UpdatedAt: %q ≠ %q", restored.UpdatedAt, original.UpdatedAt)
	}
}
