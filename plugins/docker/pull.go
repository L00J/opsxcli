package docker

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"opsxcli/internal/logger"
)

// PullOptions docker pull 配置选项
type PullOptions struct {
	Images      []string // 镜像列表
	Registries  []string // 镜像源列表
	Concurrency int      // 并发数
}

// 默认镜像源列表（国内加速源）
var defaultRegistries = []string{
	// 高速镜像源
	"docker.aityp.com",                          // AI TYP 镜像
	"docker.1ms.run",                            // 1ms 镜像
	"docker.m.daocloud.io",                      // DaoCloud 镜像
	"mirror.ccs.tencentyun.com",                 // 腾讯云镜像
	// 国内大学镜像源
	"docker.mirrors.sjtug.sjtu.edu.cn",          // 上海交大
	"docker.nju.edu.cn",                         // 南京大学
	"docker.mirrors.ustc.edu.cn",                // 中科大
	// 其他镜像源
	"dockerproxy.com",                           // Docker Proxy
	"docker.xuanyuan.me",                        // 轩辕镜像
	"docker.1panel.live",                        // 1Panel 镜像
	"docker-0.unsee.tech",                       // Unsee 镜像
	"hub-mirror.c.163.com",                      // 网易镜像
	// 官方源（备用）
	"docker.io",                                 // Docker Hub
	"registry.cn-hangzhou.aliyuncs.com",         // 阿里云
}

// Pull 拉取 Docker 镜像（自动智能加速）
func Pull(opts *PullOptions) error {
	// 如果没有指定镜像源，使用默认列表
	if len(opts.Registries) == 0 {
		opts.Registries = defaultRegistries
	}

	// 检查 docker 是否已安装
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	logger.Info("开始拉取 %d 个镜像，并发数: %d", len(opts.Images), opts.Concurrency)

	// 智能测速，选择最快的源（后台静默处理）
	ctx := context.Background()

	// 先选择最快的 5 个源作为主力
	fastRegistries := selectFastRegistries(ctx, opts.Registries, 5)

	// 如果可用源太少，使用全部
	var allRegistries []string
	if len(fastRegistries) < 3 {
		allRegistries = opts.Registries
	} else {
		// 构建完整列表：快速源 + 其他源（作为备用）
		remainingMap := make(map[string]bool)
		for _, reg := range opts.Registries {
			remainingMap[reg] = true
		}

		allRegistries = make([]string, 0, len(opts.Registries))
		// 先加入快速源
		for _, reg := range fastRegistries {
			allRegistries = append(allRegistries, reg)
			delete(remainingMap, reg)
		}
		// 再加入剩余源作为备用
		for reg := range remainingMap {
			allRegistries = append(allRegistries, reg)
		}
	}

	// 使用 goroutine 并发下载镜像
	semaphore := make(chan struct{}, opts.Concurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(opts.Images))

	startTime := time.Now()

	for _, image := range opts.Images {
		wg.Add(1)
		go func(img string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := pullSingleImageWithFallback(ctx, img, allRegistries); err != nil {
				errChan <- fmt.Errorf("拉取镜像 %s 失败: %v", img, err)
			}
		}(image)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	duration := time.Since(startTime)

	if len(errors) > 0 {
		logger.Error("拉取完成，但有 %d 个镜像失败 (耗时: %v):", len(errors), duration)
		for _, err := range errors {
			logger.Error("  - %v", err)
		}
		return fmt.Errorf("部分镜像拉取失败")
	}

	logger.Success("所有镜像拉取完成！总耗时: %v", duration)
	return nil
}

// selectFastRegistries 快速选择可用的镜像源
func selectFastRegistries(ctx context.Context, registries []string, count int) []string {
	// 静默测速，不输出信息
	type result struct {
		registry string
		latency  time.Duration
		success  bool
	}

	results := make(chan result, len(registries))
	var wg sync.WaitGroup

	testCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	for _, registry := range registries {
		wg.Add(1)
		go func(reg string) {
			defer wg.Done()

			start := time.Now()
			success := testRegistry(testCtx, reg)
			latency := time.Since(start)

			results <- result{
				registry: reg,
				latency:  latency,
				success:  success,
			}
		}(registry)
	}

	wg.Wait()
	close(results)

	// 收集结果并排序
	var successResults []result
	for r := range results {
		if r.success {
			successResults = append(successResults, r)
		}
	}

	// 按延迟排序
	sort.Slice(successResults, func(i, j int) bool {
		return successResults[i].latency < successResults[j].latency
	})

	// 选择最快的 N 个
	if count > len(successResults) {
		count = len(successResults)
	}

	selected := make([]string, count)
	for i := 0; i < count; i++ {
		selected[i] = successResults[i].registry
	}

	return selected
}

// testRegistry 简单测试镜像源是否可用
func testRegistry(ctx context.Context, registry string) bool {
	// 构建测试 URL
	url := fmt.Sprintf("https://%s/v2/", registry)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 200 或 401 都表示服务可用（401 表示需要认证，但服务在线）
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

// pullSingleImageWithFallback 拉取单个镜像（支持多源容错）
func pullSingleImageWithFallback(ctx context.Context, image string, registries []string) error {
	logger.Info("[%s] 开始拉取镜像...", image)

	var lastErr error
	var attemptCount int

	// 尝试所有镜像源
	for i, registry := range registries {
		// 构建完整镜像名
		fullImage := buildProxyImageName(registry, image)

		// 只在重试时显示源信息
		if i > 0 {
			logger.Debug("[%s] 切换到备用源 %s (%d/%d)", image, registry, i+1, len(registries))
		}

		attemptCount++

		// 显示进度
		if err := execDockerPullWithProgress(ctx, fullImage, image); err != nil {
			lastErr = err

			// 如果还有备用源，继续尝试
			if i < len(registries)-1 {
				continue
			}
			break
		}

		// 如果原始镜像名和完整镜像名不同，打标签
		if fullImage != image {
			if err := tagImage(fullImage, image); err != nil {
				logger.Debug("[%s] 打标签失败: %v", image, err)
			}
		}

		logger.Success("[%s] 拉取成功！", image)
		return nil
	}

	return fmt.Errorf("尝试了 %d 个镜像源均失败，最后错误: %v", attemptCount, lastErr)
}

// buildProxyImageName 构建代理镜像名称
func buildProxyImageName(registry, image string) string {
	// 如果镜像已经包含 registry，直接返回
	if strings.Contains(image, "/") && strings.Contains(strings.Split(image, "/")[0], ".") {
		return image
	}

	// 对于 Docker Hub 官方镜像（如 nginx, redis）
	// 需要添加 library 前缀
	if !strings.Contains(image, "/") {
		// nginx:latest -> library/nginx:latest
		image = "library/" + image
	}

	// 构建完整镜像名
	// registry/library/nginx:latest
	return registry + "/" + image
}

// execDockerPullWithProgress 执行 docker pull 命令（带进度显示）
func execDockerPullWithProgress(ctx context.Context, fullImage, displayImage string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", fullImage)

	// 创建管道获取实时输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// 读取 stdout - 显示关键进度信息
	done := make(chan bool, 2)

	// 处理输出的通用函数
	processOutput := func(scanner *bufio.Scanner) {
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// 显示关键信息
			if strings.Contains(line, "Pulling from") {
				fmt.Printf("  %s\n", line)
			} else if strings.Contains(line, "Waiting") {
				// 静默处理 Waiting
			} else if strings.Contains(line, "Downloading") {
				// 提取下载进度
				fmt.Printf("\r  %s", line)
			} else if strings.Contains(line, "Extracting") {
				// 提取解压进度
				fmt.Printf("\r  %s", line)
			} else if strings.Contains(line, "Pull complete") {
				fmt.Printf("\r  %s\n", line)
			} else if strings.Contains(line, "Digest:") || strings.Contains(line, "Status:") {
				// 最终状态信息
				fmt.Printf("  %s\n", line)
			} else if strings.Contains(line, "Error") || strings.Contains(line, "error") {
				logger.Debug("  错误: %s", line)
			}
		}
		done <- true
	}

	go processOutput(bufio.NewScanner(stdout))
	go processOutput(bufio.NewScanner(stderr))

	// 等待命令完成
	err = cmd.Wait()

	// 等待两个输出处理完成
	<-done
	<-done

	// 清除进度行
	fmt.Print("\r\033[K")

	return err
}

// tagImage 给镜像打标签
func tagImage(source, target string) error {
	cmd := exec.Command("docker", "tag", source, target)
	return cmd.Run()
}

// checkDockerInstalled 检查 docker 是否已安装
func checkDockerInstalled() error {
	cmd := exec.Command("docker", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker 未安装或不在 PATH 中，请先安装 docker")
	}
	return nil
}

// getCacheDir 获取缓存目录
func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, ".opsxcli", "docker-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}

	return cacheDir, nil
}
