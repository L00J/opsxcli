package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"opsxcli/internal/logger"
)

// ImageInfo 镜像基本信息
type ImageInfo struct {
	ID          string            `json:"Id"`
	RepoTags    []string          `json:"RepoTags,omitempty"`
	RepoDigests []string          `json:"RepoDigests,omitempty"`
	Created     int64             `json:"Created"`
	Size        int64             `json:"Size"`
	Labels      map[string]string `json:"Labels,omitempty"`
	ParentID    string            `json:"ParentId,omitempty"`
}

// ImageInspectResult 镜像详细信息
type ImageInspectResult struct {
	ID              string
	RepoTags        []string
	RepoDigests     []string
	Created         time.Time
	Author          string
	Architecture    string
	OS              string
	Size            int64
	VirtualSize     int64
	Labels          map[string]string
	ContainerConfig ContainerConfigInfo
	ExposedPorts    []string
	Env             []string
	Entrypoint      []string
	Cmd             []string
	WorkingDir      string
}

// ContainerConfigInfo 容器配置摘要
type ContainerConfigInfo struct {
	User         string
	ExposedPorts map[string]interface{}
	Env          []string
	Cmd          []string
	Entrypoint   []string
	WorkingDir   string
}

// ImagesOptions docker images 选项
type ImagesOptions struct {
	All     bool   // 显示所有镜像（包括中间层）
	Filters string // 过滤条件
	Quiet   bool   // 只显示 ID
	NoTrunc bool   // 不截断 ID
}

// RMIOptions docker rmi 选项
type RMIOptions struct {
	Images  []string // 镜像名称或 ID 列表
	Force   bool     // 强制删除
	NoPrune bool     // 不删除未标记的父镜像
}

// TagOptions docker tag 选项
type TagOptions struct {
	Source string // 源镜像
	Target string // 目标镜像名称
}

// PushOptions docker push 选项
type PushImageOptions struct {
	Image string // 镜像名称
}

// Images 列出 Docker 镜像
func Images(opts *ImagesOptions) error {
	if opts == nil {
		opts = &ImagesOptions{}
	}

	args := []string{"images", "--format", "{{json .}}"}

	if opts.All {
		args = append(args, "--all")
	}
	if opts.Filters != "" {
		args = append(args, "--filter", opts.Filters)
	}
	if opts.NoTrunc {
		args = append(args, "--no-trunc")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("没有找到镜像")
		return nil
	}

	if opts.Quiet {
		for _, line := range lines {
			var img ImageInfo
			if err := json.Unmarshal([]byte(line), &img); err == nil {
				id := img.ID
				if !opts.NoTrunc {
					if len(id) > 19 {
						id = id[:19]
					}
				}
				// 去掉 sha256: 前缀
				if strings.HasPrefix(id, "sha256:") {
					id = strings.TrimPrefix(id, "sha256:")
				}
				fmt.Println(id)
			}
		}
		return nil
	}

	return printImageTable(lines)
}

// printImageTable 格式化输出镜像列表
func printImageTable(lines []string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tTAG\tIMAGE ID\tCREATED\tSIZE")

	for _, line := range lines {
		if line == "" {
			continue
		}
		var img ImageInfo
		if err := json.Unmarshal([]byte(line), &img); err != nil {
			logger.Warning("解析镜像信息失败: %v", err)
			continue
		}

		repo, tag := parseRepoTag(img.RepoTags)
		id := formatImageID(img.ID)
		created := formatImageCreated(img.Created)
		size := formatSize(img.Size)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", repo, tag, id, created, size)
	}

	return w.Flush()
}

// parseRepoTag 解析镜像仓库和标签
func parseRepoTag(repoTags []string) (repo, tag string) {
	if len(repoTags) == 0 {
		return "<none>", "<none>"
	}

	full := repoTags[0]
	parts := strings.SplitN(full, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return full, "<none>"
}

// formatImageID 格式化镜像 ID
func formatImageID(id string) string {
	if strings.HasPrefix(id, "sha256:") {
		id = strings.TrimPrefix(id, "sha256:")
	}
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// formatImageCreated 格式化镜像创建时间
func formatImageCreated(created int64) string {
	if created == 0 {
		return "N/A"
	}
	t := time.Unix(created, 0)
	dur := time.Since(t)

	if dur < time.Minute {
		return "刚刚"
	} else if dur < time.Hour {
		return fmt.Sprintf("%d分钟前", int(dur.Minutes()))
	} else if dur < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(dur.Hours()))
	} else if dur < 30*24*time.Hour {
		return fmt.Sprintf("%d天前", int(dur.Hours()/24))
	} else if dur < 365*24*time.Hour {
		return fmt.Sprintf("%d月前", int(dur.Hours()/(24*30)))
	}
	return fmt.Sprintf("%d年前", int(dur.Hours()/(24*365)))
}

// RMI 删除 Docker 镜像
func RMI(opts *RMIOptions) error {
	if opts == nil || len(opts.Images) == 0 {
		return fmt.Errorf("请指定要删除的镜像")
	}

	args := []string{"rmi"}

	if opts.Force {
		args = append(args, "--force")
	}
	if opts.NoPrune {
		args = append(args, "--no-prune")
	}

	args = append(args, opts.Images...)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	fmt.Print(string(output))
	return nil
}

// Tag 为 Docker 镜像打标签
func Tag(opts *TagOptions) error {
	if opts == nil || opts.Source == "" || opts.Target == "" {
		return fmt.Errorf("请指定源镜像和目标标签 (source:target)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "tag", opts.Source, opts.Target)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	fmt.Printf("已标记: %s -> %s\n", opts.Source, opts.Target)
	return nil
}

// PushImage 推送 Docker 镜像
func PushImage(opts *PushImageOptions) error {
	if opts == nil || opts.Image == "" {
		return fmt.Errorf("请指定要推送的镜像")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "push", opts.Image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("推送镜像失败: %w", err)
	}

	return nil
}

// InspectImage 查看镜像详细信息
func InspectImage(image string) (*ImageInspectResult, error) {
	if image == "" {
		return nil, fmt.Errorf("请指定镜像名称或 ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s", cleanDockerError(err))
	}

	var raw []map[string]interface{}
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("解析镜像详情失败: %w", err)
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("未找到镜像: %s", image)
	}

	return parseImageInspect(raw[0]), nil
}

// parseImageInspect 解析镜像详情 JSON
func parseImageInspect(raw map[string]interface{}) *ImageInspectResult {
	result := &ImageInspectResult{
		ID:           getStr(raw, "Id"),
		Author:       getStr(raw, "Author"),
		Architecture: getStr(raw, "Architecture"),
		OS:           getStr(raw, "Os"),
		Labels:       make(map[string]string),
	}

	// RepoTags
	if rt, ok := raw["RepoTags"].([]interface{}); ok {
		for _, t := range rt {
			if s, ok := t.(string); ok {
				result.RepoTags = append(result.RepoTags, s)
			}
		}
	}

	// RepoDigests
	if rd, ok := raw["RepoDigests"].([]interface{}); ok {
		for _, d := range rd {
			if s, ok := d.(string); ok {
				result.RepoDigests = append(result.RepoDigests, s)
			}
		}
	}

	// Created
	if c := getStr(raw, "Created"); c != "" {
		if t, err := time.Parse(time.RFC3339Nano, c); err == nil {
			result.Created = t
		} else if t, err := time.Parse(time.RFC3339, c); err == nil {
			result.Created = t
		}
	}

	// Size
	result.Size = toInt64(raw["Size"])
	result.VirtualSize = toInt64(raw["VirtualSize"])

	// Labels
	if labels, ok := raw["Config"].(map[string]interface{}); ok {
		if l, ok := labels["Labels"].(map[string]interface{}); ok {
			for k, v := range l {
				if s, ok := v.(string); ok {
					result.Labels[k] = s
				}
			}
		}
	}

	// Config
	if config, ok := raw["Config"].(map[string]interface{}); ok {
		result.ContainerConfig = parseContainerConfig(config)

		// ExposedPorts
		if ep, ok := config["ExposedPorts"].(map[string]interface{}); ok {
			for port := range ep {
				result.ExposedPorts = append(result.ExposedPorts, port)
			}
		}

		// Env
		if env, ok := config["Env"].([]interface{}); ok {
			for _, e := range env {
				if s, ok := e.(string); ok {
					result.Env = append(result.Env, s)
				}
			}
		}

		// Entrypoint
		if ep, ok := config["Entrypoint"].([]interface{}); ok {
			for _, e := range ep {
				if s, ok := e.(string); ok {
					result.Entrypoint = append(result.Entrypoint, s)
				}
			}
		}

		// Cmd
		if cmd, ok := config["Cmd"].([]interface{}); ok {
			for _, c := range cmd {
				if s, ok := c.(string); ok {
					result.Cmd = append(result.Cmd, s)
				}
			}
		}

		// WorkingDir
		result.WorkingDir = getStr(config, "WorkingDir")
	}

	return result
}

// parseContainerConfig 解析容器配置
func parseContainerConfig(config map[string]interface{}) ContainerConfigInfo {
	result := ContainerConfigInfo{
		User:       getStr(config, "User"),
		WorkingDir: getStr(config, "WorkingDir"),
	}

	if ep, ok := config["ExposedPorts"].(map[string]interface{}); ok {
		result.ExposedPorts = ep
	}

	if env, ok := config["Env"].([]interface{}); ok {
		for _, e := range env {
			if s, ok := e.(string); ok {
				result.Env = append(result.Env, s)
			}
		}
	}

	if ep, ok := config["Entrypoint"].([]interface{}); ok {
		for _, e := range ep {
			if s, ok := e.(string); ok {
				result.Entrypoint = append(result.Entrypoint, s)
			}
		}
	}

	if cmd, ok := config["Cmd"].([]interface{}); ok {
		for _, c := range cmd {
			if s, ok := c.(string); ok {
				result.Cmd = append(result.Cmd, s)
			}
		}
	}

	return result
}

// PrintImageInspect 格式化输出镜像详情
func PrintImageInspect(result *ImageInspectResult) {
	fmt.Printf("镜像 ID:       %s\n", formatImageID(result.ID))
	fmt.Printf("仓库标签:      %s\n", strings.Join(result.RepoTags, ", "))
	if len(result.RepoDigests) > 0 {
		fmt.Printf("仓库摘要:      %s\n", strings.Join(result.RepoDigests, ", "))
	}
	fmt.Printf("创建时间:      %s\n", result.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("架构:          %s/%s\n", result.Architecture, result.OS)
	fmt.Printf("大小:          %s (虚拟: %s)\n", formatSize(result.Size), formatSize(result.VirtualSize))
	if result.Author != "" {
		fmt.Printf("作者:          %s\n", result.Author)
	}
	if result.WorkingDir != "" {
		fmt.Printf("工作目录:      %s\n", result.WorkingDir)
	}
	if len(result.Entrypoint) > 0 {
		fmt.Printf("入口点:        %s\n", strings.Join(result.Entrypoint, " "))
	}
	if len(result.Cmd) > 0 {
		fmt.Printf("命令:          %s\n", strings.Join(result.Cmd, " "))
	}
	if len(result.ExposedPorts) > 0 {
		fmt.Printf("暴露端口:      %s\n", strings.Join(result.ExposedPorts, ", "))
	}
	if len(result.Env) > 0 {
		fmt.Println("环境变量:")
		for _, e := range result.Env {
			fmt.Printf("  %s\n", e)
		}
	}
	if len(result.Labels) > 0 {
		fmt.Println("标签:")
		for k, v := range result.Labels {
			fmt.Printf("  %s=%s\n", k, v)
		}
	}
}

// getStr 从 map 中获取字符串值
func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// toInt64 将 interface{} 转换为 int64
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	case json.Number:
		if n, err := val.Int64(); err == nil {
			return n
		}
	case string:
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

// SaveImage 导出镜像为 tar 文件
func SaveImage(image, outputFile string) error {
	if image == "" {
		return fmt.Errorf("请指定要导出的镜像")
	}

	args := []string{"save"}
	if outputFile != "" {
		args = append(args, "-o", outputFile)
	}
	args = append(args, image)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	if outputFile != "" {
		fmt.Printf("镜像已导出到: %s\n", outputFile)
	} else {
		fmt.Print(string(output))
	}
	return nil
}

// LoadImage 从 tar 文件导入镜像
func LoadImage(inputFile string) error {
	if inputFile == "" {
		return fmt.Errorf("请指定要导入的 tar 文件")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "load", "-i", inputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	fmt.Print(string(output))
	return nil
}

// SearchImages 搜索 Docker Hub 镜像
func SearchImages(term string, limit int) error {
	if term == "" {
		return fmt.Errorf("请指定搜索关键词")
	}

	args := []string{"search", "--format", "{{.Name}}\t{{.Description}}\t{{.StarCount}}\t{{.IsOfficial}}\t{{.IsAutomated}}"}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	args = append(args, term)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Println("没有找到匹配的镜像")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tSTARS\tOFFICIAL\tAUTOMATED")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) == 5 {
			official := ""
			if parts[3] == "true" {
				official = "[OK]"
			}
			automated := ""
			if parts[4] == "true" {
				automated = "[OK]"
			}
			// 截断描述
			desc := parts[1]
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", parts[0], desc, parts[2], official, automated)
		}
	}

	return w.Flush()
}

// HistoryImages 查看镜像构建历史
func HistoryImages(image string, noTrunc bool, quiet bool) error {
	if image == "" {
		return fmt.Errorf("请指定镜像名称或 ID")
	}

	args := []string{"history"}
	if noTrunc {
		args = append(args, "--no-trunc")
	}
	if quiet {
		args = append(args, "--format", "{{.ID}}")
	} else {
		args = append(args, "--format", "{{.ID}}\t{{.CreatedBy}}\t{{.CreatedAt}}\t{{.Size}}")
	}
	args = append(args, image)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	if quiet {
		fmt.Print(string(output))
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IMAGE\tCREATED BY\tCREATED\tSIZE")

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) == 4 {
			createdBy := parts[1]
			if !noTrunc && len(createdBy) > 50 {
				createdBy = createdBy[:47] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", parts[0], createdBy, parts[2], parts[3])
		}
	}

	return w.Flush()
}

// PruneImages 清理未使用的镜像
func PruneImages(all bool) error {
	args := []string{"image", "prune", "-f"}
	if all {
		args = append(args, "--all")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", cleanDockerError(err))
	}

	fmt.Print(string(output))
	return nil
}
