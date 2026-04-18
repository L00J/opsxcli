package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"opsxcli/internal/logger"
)

// RegistrySourcesAPI 镜像源 API 配置
type RegistrySourcesAPI struct {
	Endpoint string // API 端点
	Token    string // 认证 Token
	Timeout  time.Duration
}

// RegistrySourcesResponse API 响应格式
type RegistrySourcesResponse struct {
	Success    bool     `json:"success"`
	Message    string   `json:"message,omitempty"`
	Registries []string `json:"registries"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
}

// PrivateRegistryAuth 私有仓库认证信息
type PrivateRegistryAuth struct {
	Registry string `json:"registry"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
}

// NewRegistrySourcesAPI 创建镜像源 API 客户端
func NewRegistrySourcesAPI(endpoint, token string) *RegistrySourcesAPI {
	return &RegistrySourcesAPI{
		Endpoint: endpoint,
		Token:    token,
		Timeout:  10 * time.Second,
	}
}

// FetchRegistries 从 API 获取镜像源列表
func (api *RegistrySourcesAPI) FetchRegistries(ctx context.Context) ([]string, error) {
	client := &http.Client{
		Timeout: api.Timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", api.Endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加认证 Token
	if api.Token != "" {
		req.Header.Set("Authorization", "Bearer "+api.Token)
	}

	req.Header.Set("User-Agent", "opsxcli-docker/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 返回错误: HTTP %d, %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp RegistrySourcesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API 返回失败: %s", apiResp.Message)
	}

	logger.Info("从 API 获取到 %d 个镜像源", len(apiResp.Registries))

	return apiResp.Registries, nil
}

// GetPrivateRegistryAuth 获取私有仓库认证信息
func (api *RegistrySourcesAPI) GetPrivateRegistryAuth(ctx context.Context) (*PrivateRegistryAuth, error) {
	if api.Token == "" {
		return nil, fmt.Errorf("需要认证 Token 才能访问私有仓库")
	}

	endpoint := api.Endpoint + "/auth"

	client := &http.Client{
		Timeout: api.Timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+api.Token)
	req.Header.Set("User-Agent", "opsxcli-docker/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取私有仓库认证失败: HTTP %d", resp.StatusCode)
	}

	var auth PrivateRegistryAuth
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, err
	}

	return &auth, nil
}

// LoadRegistriesWithAPI 加载镜像源（优先从 API，失败则使用默认列表）
func LoadRegistriesWithAPI(apiEndpoint, apiToken string) []string {
	if apiEndpoint == "" {
		// 没有配置 API，使用默认列表
		return defaultRegistries
	}

	api := NewRegistrySourcesAPI(apiEndpoint, apiToken)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	registries, err := api.FetchRegistries(ctx)
	if err != nil {
		logger.Error("从 API 获取镜像源失败: %v，使用默认列表", err)
		return defaultRegistries
	}

	if len(registries) == 0 {
		logger.Error("API 返回的镜像源列表为空，使用默认列表")
		return defaultRegistries
	}

	return registries
}

// LoginPrivateRegistry 登录私有仓库
func LoginPrivateRegistry(auth *PrivateRegistryAuth) error {
	// 使用 docker login 命令登录
	// exec.Command("docker", "login", "-u", auth.Username, "-p", auth.Password, auth.Registry)

	logger.Info("登录私有仓库: %s (用户: %s)", auth.Registry, auth.Username)

	// TODO: 实现 docker login 逻辑
	// 注意：密码不应该直接通过命令行参数传递，应该使用标准输入或凭据存储

	return fmt.Errorf("私有仓库登录功能开发中")
}

// Example API endpoint format:
// GET /api/v1/docker/registries
// Response:
// {
//   "success": true,
//   "registries": ["docker.aityp.com", "docker.1ms.run", ...],
//   "updated_at": "2025-12-19T15:00:00Z"
// }
//
// GET /api/v1/docker/auth
// Headers: Authorization: Bearer <token>
// Response:
// {
//   "registry": "private.example.com",
//   "username": "user@example.com",
//   "password": "xxxxx",
//   "email": "user@example.com"
// }
