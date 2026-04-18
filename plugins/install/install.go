package install

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"opsxcli/internal/logger"
)

// OSInfo 操作系统信息
type OSInfo struct {
	ID         string // 如: centos, ubuntu, debian
	VersionID  string // 如: 7, 8, 20.04
	Name       string // 如: CentOS Linux, Ubuntu
	PrettyName string // 如: CentOS Linux 7 (Core)
}

// DetectOS 检测操作系统类型
func DetectOS() (*OSInfo, error) {
	// 支持 Linux 和 macOS
	if runtime.GOOS == "darwin" {
		// macOS 使用 Homebrew
		info := &OSInfo{
			ID:         "darwin",
			Name:       "macOS",
			PrettyName: "macOS",
		}
		return info, nil
	}

	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("当前系统不支持包管理器安装（仅支持 Linux 和 macOS）")
	}

	// 尝试读取 /etc/os-release
	osReleasePath := "/etc/os-release"
	file, err := os.Open(osReleasePath)
	if err != nil {
		// 尝试其他路径
		altPaths := []string{
			"/usr/lib/os-release",
			"/etc/redhat-release", // CentOS/RHEL 旧版本
			"/etc/debian_version", // Debian
		}

		for _, path := range altPaths {
			if file, err = os.Open(path); err == nil {
				osReleasePath = path
				break
			}
		}

		if err != nil {
			return nil, fmt.Errorf("无法读取系统版本信息: %v", err)
		}
	}
	defer file.Close()

	info := &OSInfo{}
	scanner := bufio.NewScanner(file)

	// 处理 /etc/redhat-release (旧版 CentOS/RHEL)
	if strings.Contains(osReleasePath, "redhat-release") {
		scanner.Scan()
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), "centos") {
			info.ID = "centos"
			// 尝试提取版本号
			if strings.Contains(line, "7") {
				info.VersionID = "7"
			} else if strings.Contains(line, "6") {
				info.VersionID = "6"
			}
			info.PrettyName = line
			return info, nil
		}
		if strings.Contains(strings.ToLower(line), "red hat") {
			info.ID = "rhel"
			info.PrettyName = line
			return info, nil
		}
	}

	// 处理 /etc/debian_version (Debian)
	if strings.Contains(osReleasePath, "debian_version") {
		scanner.Scan()
		version := strings.TrimSpace(scanner.Text())
		info.ID = "debian"
		info.VersionID = version
		info.PrettyName = fmt.Sprintf("Debian %s", version)
		return info, nil
	}

	// 处理 /etc/os-release (标准格式)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// 移除引号
		value = strings.Trim(value, `"`)

		switch key {
		case "ID":
			info.ID = strings.ToLower(value)
		case "VERSION_ID":
			info.VersionID = value
		case "NAME":
			info.Name = value
		case "PRETTY_NAME":
			info.PrettyName = value
		}
	}

	if info.ID == "" {
		return nil, fmt.Errorf("无法识别操作系统类型")
	}

	return info, nil
}

// GetPackageManager 根据操作系统获取包管理器命令
func GetPackageManager(osInfo *OSInfo) (string, []string, error) {
	id := osInfo.ID
	prettyName := strings.ToLower(osInfo.PrettyName)
	name := strings.ToLower(osInfo.Name)

	// macOS 使用 Homebrew
	if id == "darwin" {
		// 检查 brew 是否存在
		if _, err := exec.LookPath("brew"); err != nil {
			return "", nil, fmt.Errorf("未找到 Homebrew，请先安装: /bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
		}
		return "brew", []string{"install"}, nil
	}

	// 处理阿里云 Linux（优先检查，因为可能 ID 是 alios 或其他）
	if strings.Contains(prettyName, "aliyun") ||
		strings.Contains(name, "aliyun") ||
		strings.Contains(id, "alios") ||
		strings.Contains(id, "aliyun") {
		// 阿里云 Linux 3/4 使用 dnf，2 使用 yum
		versionID := osInfo.VersionID
		if versionID == "3" || versionID == "4" || strings.HasPrefix(versionID, "3.") || strings.HasPrefix(versionID, "4.") {
			return "dnf", []string{"install"}, nil
		}
		// 阿里云 Linux 2 使用 yum
		return "yum", []string{"install"}, nil
	}

	// 根据 ID 判断
	switch id {
	case "centos", "rhel", "rocky", "almalinux", "amzn":
		// CentOS/RHEL/Rocky/Alma/Amazon Linux 使用 yum
		return "yum", []string{"install"}, nil
	case "fedora":
		// Fedora 使用 dnf
		return "dnf", []string{"install"}, nil
	case "ubuntu", "debian":
		// Ubuntu/Debian 使用 apt-get
		return "apt-get", []string{"install"}, nil
	default:
		return "", nil, fmt.Errorf("不支持的操作系统: %s (%s)", id, osInfo.PrettyName)
	}
}

// Install 安装软件包
func Install(packageName string) error {
	// 检测操作系统
	osInfo, err := DetectOS()
	if err != nil {
		return err
	}

	logger.Info("检测到操作系统: %s", osInfo.PrettyName)

	// 获取包管理器
	pmCmd, pmArgs, err := GetPackageManager(osInfo)
	if err != nil {
		return err
	}

	logger.Info("使用包管理器: %s", pmCmd)

	// 构建完整命令
	args := append(pmArgs, packageName)
	cmd := exec.Command(pmCmd, args...)

	// 设置输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// 执行安装
	logger.Info("执行命令: %s %s", pmCmd, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装失败: %v", err)
	}

	logger.Success("安装完成: %s", packageName)
	return nil
}
