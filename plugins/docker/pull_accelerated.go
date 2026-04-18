package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"opsxcli/internal/logger"
)

// PullWithAcceleration 使用加速模式拉取镜像
// 这是最优方案：使用 Registry API 多源并行下载
func PullWithAcceleration(ctx context.Context, image string, registries []string, cacheDir string) error {
	logger.Info("使用加速模式拉取镜像: %s", image)

	// 解析镜像名和标签
	imageName, tag := parseImageAndTag(image)

	// 智能选择最优镜像源
	optimalRegistries := GetOptimalRegistries(ctx, registries, 3)
	if len(optimalRegistries) == 0 {
		return fmt.Errorf("没有可用的镜像源")
	}

	// 创建多源下载器
	downloader := NewMultiSourceDownloader(optimalRegistries, cacheDir)

	// 获取镜像 manifest（尝试多个源）
	var manifest *Manifest
	var err error
	var successRegistry string

	for _, registry := range optimalRegistries {
		client := NewRegistryClient(registry)
		manifest, err = client.GetManifest(ctx, imageName, tag)
		if err == nil {
			successRegistry = registry
			break
		}
		logger.Debug("从 %s 获取 manifest 失败: %v", registry, err)
	}

	if manifest == nil {
		return fmt.Errorf("无法获取镜像 manifest: %v", err)
	}

	logger.Success("从 %s 获取 manifest 成功", successRegistry)
	logger.Info("镜像包含 %d 个层", len(manifest.Layers))

	// 创建进度跟踪器
	progress := NewProgressTracker()

	// 启动进度显示
	stopProgress := progress.StartProgressDisplay(500 * time.Millisecond)
	defer func() {
		stopProgress <- struct{}{}
	}()

	// 下载所有层
	layerFiles := make([]string, 0, len(manifest.Layers))

	for i, layer := range manifest.Layers {
		logger.Info("\n[%d/%d] 下载层: %s", i+1, len(manifest.Layers), shortDigest(layer.Digest))

		result, err := downloader.DownloadLayer(ctx, imageName, layer, progress)
		if err != nil {
			return fmt.Errorf("下载层失败: %v", err)
		}

		layerFiles = append(layerFiles, result.FilePath)
	}

	// 下载配置文件
	logger.Info("\n下载配置文件...")
	configFile, err := downloadConfig(ctx, downloader, imageName, manifest.Config)
	if err != nil {
		return fmt.Errorf("下载配置文件失败: %v", err)
	}

	// 导入镜像到 Docker
	logger.Info("\n导入镜像到 Docker...")
	if err := importImageToDocker(image, manifest, configFile, layerFiles, cacheDir); err != nil {
		return fmt.Errorf("导入镜像失败: %v", err)
	}

	logger.Success("\n镜像拉取完成: %s", image)
	return nil
}

// downloadConfig 下载配置文件
func downloadConfig(ctx context.Context, downloader *MultiSourceDownloader, image string, config DescriptorConfig) (string, error) {
	// 将配置当作特殊的 layer 下载
	layer := LayerDescriptor{
		MediaType: config.MediaType,
		Size:      config.Size,
		Digest:    config.Digest,
	}

	result, err := downloader.DownloadLayer(ctx, image, layer, nil)
	if err != nil {
		return "", err
	}

	return result.FilePath, nil
}

// importImageToDocker 将下载的层导入到 Docker
func importImageToDocker(imageName string, manifest *Manifest, configFile string, layerFiles []string, cacheDir string) error {
	// 创建临时目录用于构建 Docker 镜像包
	tempDir := filepath.Join(cacheDir, "import-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// 方式 1: 使用 docker load (需要构建符合 Docker 格式的 tar)
	// 这部分实现比较复杂，需要构建 manifest.json, repositories 等
	// 简化实现：先将文件复制到临时目录，然后用 docker import

	// 方式 2: 使用已有的 docker pull 作为 fallback
	// 如果直接导入太复杂，可以先保存层到 /var/lib/docker/image/overlay2/layerdb
	// 但这需要 root 权限且与 Docker 内部实现耦合

	// 实用方案：将下载的层打包成 tar，然后使用 docker load
	tarFile := filepath.Join(tempDir, "image.tar")
	if err := createDockerImageTar(tarFile, manifest, configFile, layerFiles); err != nil {
		return err
	}

	// 使用 docker load 导入
	cmd := exec.Command("docker", "load", "-i", tarFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker load 失败: %v", err)
	}

	// 打标签
	if err := tagLoadedImage(manifest, imageName); err != nil {
		logger.Error("打标签失败: %v (镜像已导入，可手动打标签)", err)
	}

	return nil
}

// createDockerImageTar 创建 Docker 镜像 tar 包
func createDockerImageTar(tarFile string, manifest *Manifest, configFile string, layerFiles []string) error {
	// TODO: 实现完整的 Docker 镜像 tar 格式
	// 需要包含：
	// 1. manifest.json - 镜像清单
	// 2. <digest>.json - 配置文件
	// 3. <digest>/layer.tar - 每个层
	// 4. repositories - 仓库信息

	// 简化实现：这里返回错误，提示用户使用快速模式
	return fmt.Errorf("docker load 格式构建功能开发中，请使用快速模式（--mode=fast）")
}

// tagLoadedImage 给导入的镜像打标签
func tagLoadedImage(manifest *Manifest, targetName string) error {
	// 计算镜像 ID（配置文件的 digest）
	imageID := manifest.Config.Digest

	cmd := exec.Command("docker", "tag", imageID, targetName)
	return cmd.Run()
}

// parseImageAndTag 解析镜像名和标签
func parseImageAndTag(image string) (string, string) {
	// 去除协议前缀
	image = strings.TrimPrefix(image, "http://")
	image = strings.TrimPrefix(image, "https://")

	// 分离标签
	parts := strings.Split(image, ":")
	if len(parts) == 1 {
		return image, "latest"
	}

	// 处理包含 digest 的情况 (image@sha256:xxx)
	if strings.Contains(image, "@") {
		parts = strings.Split(image, "@")
		return parts[0], parts[1]
	}

	// 标准格式 (image:tag)
	return parts[0], parts[1]
}
