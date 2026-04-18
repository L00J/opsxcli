package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RegistryClient Docker Registry V2 API 客户端
type RegistryClient struct {
	Registry   string
	HTTPClient *http.Client
	Token      string
}

// Manifest 镜像清单
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	MediaType     string            `json:"mediaType"`
	Config        DescriptorConfig  `json:"config"`
	Layers        []LayerDescriptor `json:"layers"`
}

// DescriptorConfig 配置描述符
type DescriptorConfig struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// LayerDescriptor 层描述符
type LayerDescriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// NewRegistryClient 创建 Registry 客户端
func NewRegistryClient(registry string) *RegistryClient {
	return &RegistryClient{
		Registry: registry,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetManifest 获取镜像清单
func (c *RegistryClient) GetManifest(ctx context.Context, image, tag string) (*Manifest, error) {
	// 构建 API URL
	url := c.buildManifestURL(image, tag)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置 Accept 头（支持多种 manifest 格式）
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")

	// 如果有 token，添加到请求头
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	// 发送请求
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取 manifest 失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode == http.StatusUnauthorized {
		// 需要认证，尝试获取 token
		if err := c.authenticate(resp, image); err != nil {
			return nil, err
		}
		// 重试请求
		return c.GetManifest(ctx, image, tag)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取 manifest 失败: HTTP %d, %s", resp.StatusCode, string(body))
	}

	// 解析 manifest
	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("解析 manifest 失败: %v", err)
	}

	return &manifest, nil
}

// DownloadLayer 下载单个层（支持 Range 请求）
func (c *RegistryClient) DownloadLayer(ctx context.Context, image string, digest string, start, end int64) ([]byte, error) {
	url := c.buildBlobURL(image, digest)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 设置 Range 头（分片下载）
	if start >= 0 && end > start {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查认证
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.authenticate(resp, image); err != nil {
			return nil, err
		}
		return c.DownloadLayer(ctx, image, digest, start, end)
	}

	// 206 Partial Content 或 200 OK 都是成功
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("下载层失败: HTTP %d", resp.StatusCode)
	}

	// 读取数据
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取数据失败: %v", err)
	}

	return data, nil
}

// authenticate 认证获取 token
func (c *RegistryClient) authenticate(resp *http.Response, image string) error {
	// 解析 WWW-Authenticate 头
	authHeader := resp.Header.Get("Www-Authenticate")
	if authHeader == "" {
		return fmt.Errorf("需要认证但未提供认证信息")
	}

	// 简单解析 Bearer token 认证
	// 格式: Bearer realm="...",service="...",scope="..."
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return fmt.Errorf("不支持的认证方式: %s", authHeader)
	}

	// 解析认证参数
	params := parseAuthParams(authHeader[7:]) // 去掉 "Bearer "

	realm := params["realm"]
	service := params["service"]
	scope := params["scope"]

	if realm == "" {
		return fmt.Errorf("认证 realm 为空")
	}

	// 构建认证 URL
	authURL := realm + "?service=" + service
	if scope != "" {
		authURL += "&scope=" + scope
	}

	// 发送认证请求
	authReq, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return err
	}

	authResp, err := c.HTTPClient.Do(authReq)
	if err != nil {
		return fmt.Errorf("认证请求失败: %v", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		return fmt.Errorf("认证失败: HTTP %d", authResp.StatusCode)
	}

	// 解析 token
	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(authResp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("解析认证响应失败: %v", err)
	}

	// 保存 token
	if tokenResp.Token != "" {
		c.Token = tokenResp.Token
	} else if tokenResp.AccessToken != "" {
		c.Token = tokenResp.AccessToken
	}

	return nil
}

// buildManifestURL 构建 manifest API URL
func (c *RegistryClient) buildManifestURL(image, tag string) string {
	// 处理镜像名（移除可能的 registry 前缀）
	imageName := c.normalizeImageName(image)

	// Docker Hub 需要特殊处理
	if c.Registry == "docker.io" || c.Registry == "registry-1.docker.io" {
		return fmt.Sprintf("https://registry-1.docker.io/v2/%s/manifests/%s", imageName, tag)
	}

	// 其他 registry
	return fmt.Sprintf("https://%s/v2/%s/manifests/%s", c.Registry, imageName, tag)
}

// buildBlobURL 构建 blob API URL
func (c *RegistryClient) buildBlobURL(image, digest string) string {
	imageName := c.normalizeImageName(image)

	if c.Registry == "docker.io" || c.Registry == "registry-1.docker.io" {
		return fmt.Sprintf("https://registry-1.docker.io/v2/%s/blobs/%s", imageName, digest)
	}

	return fmt.Sprintf("https://%s/v2/%s/blobs/%s", c.Registry, imageName, digest)
}

// normalizeImageName 规范化镜像名
func (c *RegistryClient) normalizeImageName(image string) string {
	// 移除可能的 registry 前缀
	parts := strings.Split(image, "/")

	// 如果第一部分包含 '.'，说明是 registry 地址，需要移除
	if len(parts) > 1 && strings.Contains(parts[0], ".") {
		return strings.Join(parts[1:], "/")
	}

	// Docker Hub 官方镜像需要添加 library 前缀
	if c.Registry == "docker.io" || c.Registry == "registry-1.docker.io" {
		if len(parts) == 1 {
			return "library/" + image
		}
	}

	return image
}

// parseAuthParams 解析认证参数
func parseAuthParams(authStr string) map[string]string {
	params := make(map[string]string)

	// 按逗号分割
	pairs := strings.Split(authStr, ",")

	for _, pair := range pairs {
		// 按等号分割
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		// 移除引号
		value = strings.Trim(value, `"`)

		params[key] = value
	}

	return params
}

// Ping 测试 registry 连通性和延迟
func (c *RegistryClient) Ping(ctx context.Context) (time.Duration, error) {
	url := fmt.Sprintf("https://%s/v2/", c.Registry)

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	// 200 或 401 都表示服务可用
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		return latency, fmt.Errorf("registry 不可用: HTTP %d", resp.StatusCode)
	}

	return latency, nil
}
