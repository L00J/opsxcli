package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"opsxcli/internal/logger"
)

// ComposeUpOptions docker compose up 选项
type ComposeUpOptions struct {
	File     string   // 指定 compose 文件 (-f)
	Project  string   // 项目名称 (-p)
	Build    bool     // 构建镜像后再启动 (--build)
	Detach   bool     // 后台运行 (-d)
	Force    bool     // 强制重建 (--force-recreate)
	NoStart  bool     // 只创建不启动 (--no-start)
	Quiet    bool     // 静默模式 (-q)
	Services []string // 指定启动的服务
	Remove   bool     // 启动前移除孤立容器 (--remove-orphans)
}

// ComposeDownOptions docker compose down 选项
type ComposeDownOptions struct {
	File          string // 指定 compose 文件 (-f)
	Project       string // 项目名称 (-p)
	RemoveOrphans bool   // 移除孤立容器 (--remove-orphans)
	Volumes       bool   // 删除卷 (--volumes)
	Images        string // 删除镜像类型 (--rmi, all/local)
	Timeout       int    // 超时秒数 (--timeout)
}

// ComposePSOptions docker compose ps 选项
type ComposePSOptions struct {
	File    string // 指定 compose 文件 (-f)
	Project string // 项目名称 (-p)
	All     bool   // 显示所有服务 (-a)
	Quiet   bool   // 只显示 ID (-q)
	Format  string // 输出格式 (--format)
}

// ComposeLogsOptions docker compose logs 选项
type ComposeLogsOptions struct {
	File       string   // 指定 compose 文件 (-f)
	Project    string   // 项目名称 (-p)
	Follow     bool     // 持续输出 (-f)
	Tail       string   // 显示最后 N 行 (--tail)
	Since      string   // 显示自此时间后的日志 (--since)
	Until      string   // 显示至此时间前的日志 (--until)
	Timestamps bool     // 显示时间戳 (-t)
	Services   []string // 指定服务
}

// ComposeBuildOptions docker compose build 选项
type ComposeBuildOptions struct {
	File     string   // 指定 compose 文件 (-f)
	Project  string   // 项目名称 (-p)
	NoCache  bool     // 不使用缓存 (--no-cache)
	Pull     bool     // 始终拉取最新镜像 (--pull)
	Parallel bool     // 并行构建 (--parallel)
	Quiet    bool     // 静默模式 (-q)
	Services []string // 指定构建的服务
}

// ComposePullOptions docker compose pull 选项
type ComposePullOptions struct {
	File           string   // 指定 compose 文件 (-f)
	Project        string   // 项目名称 (-p)
	Quiet          bool     // 静默模式 (-q)
	IgnoreFailures bool     // 忽略拉取失败 (--ignore-build-failures)
	Services       []string // 指定拉取的服务
}

// ComposeRestartOptions docker compose restart 选项
type ComposeRestartOptions struct {
	File     string   // 指定 compose 文件 (-f)
	Project  string   // 项目名称 (-p)
	Timeout  int      // 超时秒数 (--timeout)
	Services []string // 指定重启的服务
}

// ComposeStopOptions docker compose stop 选项
type ComposeStopOptions struct {
	File     string   // 指定 compose 文件 (-f)
	Project  string   // 项目名称 (-p)
	Timeout  int      // 超时秒数 (--timeout)
	Services []string // 指定停止的服务
}

// =============================================================================
// Compose Up
// =============================================================================

// ComposeUp 启动 Docker Compose 服务
func ComposeUp(opts *ComposeUpOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "up")

	if opts.Detach {
		args = append(args, "-d")
	}
	if opts.Build {
		args = append(args, "--build")
	}
	if opts.Force {
		args = append(args, "--force-recreate")
	}
	if opts.NoStart {
		args = append(args, "--no-start")
	}
	if opts.Quiet {
		args = append(args, "-q")
	}
	if opts.Remove {
		args = append(args, "--remove-orphans")
	}

	// 指定服务
	args = append(args, opts.Services...)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose up 失败: %v", cleanDockerError(err))
	}

	logger.Info("docker compose up 执行完成")
	return nil
}

// =============================================================================
// Compose Down
// =============================================================================

// ComposeDown 停止并移除 Docker Compose 服务
func ComposeDown(opts *ComposeDownOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "down")

	if opts.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if opts.Volumes {
		args = append(args, "--volumes")
	}
	if opts.Images != "" {
		args = append(args, "--rmi", opts.Images)
	}
	if opts.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", opts.Timeout))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose down 失败: %v", cleanDockerError(err))
	}

	logger.Info("docker compose down 执行完成")
	return nil
}

// =============================================================================
// Compose PS
// =============================================================================

// ComposePS 列出 Docker Compose 服务
func ComposePS(opts *ComposePSOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "ps")

	if opts.All {
		args = append(args, "-a")
	}
	if opts.Quiet {
		args = append(args, "-q")
	}
	if opts.Format != "" {
		args = append(args, "--format", opts.Format)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose ps 失败: %v", cleanDockerError(err))
	}

	return nil
}

// =============================================================================
// Compose Logs
// =============================================================================

// ComposeLogs 查看 Docker Compose 服务日志
func ComposeLogs(opts *ComposeLogsOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "logs")

	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Tail != "" {
		args = append(args, "--tail", opts.Tail)
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}
	if opts.Timestamps {
		args = append(args, "-t")
	}

	// 指定服务
	args = append(args, opts.Services...)

	timeout := 60
	if opts.Follow {
		timeout = 3600 // 跟随模式给更长时间
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose logs 失败: %v", cleanDockerError(err))
	}

	return nil
}

// =============================================================================
// Compose Build
// =============================================================================

// ComposeBuild 构建 Docker Compose 服务
func ComposeBuild(opts *ComposeBuildOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "build")

	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	if opts.Parallel {
		args = append(args, "--parallel")
	}
	if opts.Quiet {
		args = append(args, "-q")
	}

	// 指定服务
	args = append(args, opts.Services...)

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose build 失败: %v", cleanDockerError(err))
	}

	logger.Info("docker compose build 执行完成")
	return nil
}

// =============================================================================
// Compose Pull
// =============================================================================

// ComposePull 拉取 Docker Compose 服务镜像
func ComposePull(opts *ComposePullOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "pull")

	if opts.Quiet {
		args = append(args, "-q")
	}
	if opts.IgnoreFailures {
		args = append(args, "--ignore-build-failures")
	}

	// 指定服务
	args = append(args, opts.Services...)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose pull 失败: %v", cleanDockerError(err))
	}

	logger.Info("docker compose pull 执行完成")
	return nil
}

// =============================================================================
// Compose Restart
// =============================================================================

// ComposeRestart 重启 Docker Compose 服务
func ComposeRestart(opts *ComposeRestartOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "restart")

	if opts.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", opts.Timeout))
	}

	// 指定服务
	args = append(args, opts.Services...)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose restart 失败: %v", cleanDockerError(err))
	}

	logger.Info("docker compose restart 执行完成")
	return nil
}

// =============================================================================
// Compose Stop
// =============================================================================

// ComposeStop 停止 Docker Compose 服务
func ComposeStop(opts *ComposeStopOptions) error {
	if err := checkDockerInstalled(); err != nil {
		return err
	}

	if opts == nil {
		return fmt.Errorf("请指定 compose 选项")
	}

	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "stop")

	if opts.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", opts.Timeout))
	}

	// 指定服务
	args = append(args, opts.Services...)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose stop 失败: %v", cleanDockerError(err))
	}

	logger.Info("docker compose stop 执行完成")
	return nil
}

// =============================================================================
// 辅助函数（纯函数，可测试）
// =============================================================================

// appendComposeFile 添加 compose 文件参数
func appendComposeFile(args []string, file string) []string {
	if file != "" {
		args = append(args, "-f", file)
	}
	return args
}

// appendComposeProject 添加项目名称参数
func appendComposeProject(args []string, project string) []string {
	if project != "" {
		args = append(args, "-p", project)
	}
	return args
}

// buildComposeUpArgs 构建 compose up 命令参数（纯函数，可测试）
func buildComposeUpArgs(opts *ComposeUpOptions) []string {
	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "up")

	if opts.Detach {
		args = append(args, "-d")
	}
	if opts.Build {
		args = append(args, "--build")
	}
	if opts.Force {
		args = append(args, "--force-recreate")
	}
	if opts.NoStart {
		args = append(args, "--no-start")
	}
	if opts.Quiet {
		args = append(args, "-q")
	}
	if opts.Remove {
		args = append(args, "--remove-orphans")
	}

	args = append(args, opts.Services...)
	return args
}

// buildComposeDownArgs 构建 compose down 命令参数（纯函数，可测试）
func buildComposeDownArgs(opts *ComposeDownOptions) []string {
	args := []string{"compose"}
	args = appendComposeFile(args, opts.File)
	args = appendComposeProject(args, opts.Project)
	args = append(args, "down")

	if opts.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if opts.Volumes {
		args = append(args, "--volumes")
	}
	if opts.Images != "" {
		args = append(args, "--rmi", opts.Images)
	}
	if opts.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", opts.Timeout))
	}

	return args
}

// cleanDockerError is defined in container.go
