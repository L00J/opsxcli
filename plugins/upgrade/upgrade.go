package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

const (
	giteeReleaseURL = "https://gitee.com/opsx-tools/opsxcli/releases/download/latest/opsxcli-%s-%s.tar.gz"
	userAgent       = "opsxcli-upgrade"
	maxBackups      = 2 // 最多保留的备份数量
)

// Run 执行升级操作
func Run() error {
	fmt.Println("🚀 开始升级 opsxcli...")

	// 获取当前二进制文件路径
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取当前程序路径失败: %w", err)
	}

	// 解析符号链接，获取真实路径
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("解析程序路径失败: %w", err)
	}

	// 智能检测安装位置
	targetBinary := detectInstallPath(currentBinary)
	if targetBinary != currentBinary {
		fmt.Printf("📍 检测到系统安装: %s\n", targetBinary)
	}

	// 构建下载URL（转换为 Gitee 文件名格式）
	osName := convertOSName(runtime.GOOS)
	arch := convertArch(runtime.GOOS, runtime.GOARCH)
	downloadURL := fmt.Sprintf(giteeReleaseURL, osName, arch)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "opsxcli-upgrade-*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 下载文件
	tarGzPath := filepath.Join(tempDir, "opsxcli.tar.gz")
	if err := downloadFile(downloadURL, tarGzPath); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	// 解压文件
	extractedPath := filepath.Join(tempDir, "opsxcli")
	if err := extractTarGz(tarGzPath, tempDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 验证解压后的文件
	if _, err := os.Stat(extractedPath); os.IsNotExist(err) {
		return fmt.Errorf("解压后的文件不存在: %s", extractedPath)
	}

	// 备份当前二进制文件
	backupPath := targetBinary + ".backup." + time.Now().Format("20060102150405")
	if err := copyFile(targetBinary, backupPath); err != nil {
		return fmt.Errorf("备份失败: %w", err)
	}

	// 替换二进制文件
	if err := replaceFile(extractedPath, targetBinary); err != nil {
		// 替换失败，尝试恢复备份
		fmt.Println("⚠️  替换失败，尝试恢复备份...")
		if restoreErr := copyFile(backupPath, targetBinary); restoreErr != nil {
			return fmt.Errorf("替换失败且恢复备份失败: %w, 恢复错误: %v", err, restoreErr)
		}
		return fmt.Errorf("替换失败（已恢复备份）: %w", err)
	}

	fmt.Println("✅ 升级成功！")

	// 清理旧备份，只保留最近的3个（静默处理）
	cleanOldBackups(targetBinary)

	return nil
}

// detectInstallPath 智能检测安装路径
// 优先级：/usr/local/bin > /usr/bin > 当前路径
func detectInstallPath(currentPath string) string {
	// 标准安装路径列表（按优先级排序）
	standardPaths := []string{
		"/usr/local/bin/opsxcli",
		"/usr/bin/opsxcli",
	}

	// 如果当前路径已经是标准路径之一，直接使用
	for _, path := range standardPaths {
		if currentPath == path {
			return currentPath
		}
	}

	// 检查标准路径是否存在可执行文件
	for _, path := range standardPaths {
		if info, err := os.Stat(path); err == nil {
			// 检查是否是普通文件且可执行
			if !info.IsDir() && info.Mode()&0111 != 0 {
				return path
			}
		}
	}

	// 都不存在，使用当前路径
	return currentPath
}

// convertOSName 转换 runtime.GOOS 到 Gitee 文件名格式
func convertOSName(goos string) string {
	// Gitee 文件名格式：首字母大写
	// linux -> Linux
	// darwin -> Darwin
	// windows -> Windows
	if goos == "" {
		return ""
	}
	// 手动实现首字母大写
	return strings.ToUpper(goos[:1]) + goos[1:]
}

// convertArch 转换 runtime.GOARCH 到 Gitee 文件名格式
func convertArch(goos, goarch string) string {
	// Gitee 文件名格式：
	// Linux amd64 -> x86_64
	// Linux arm64 -> aarch64
	// Darwin amd64 -> x86_64
	// Darwin arm64 -> arm64
	// Windows amd64 -> x86_64
	// 386 -> i386
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		// Linux 上 arm64 对应 aarch64，macOS 上对应 arm64
		if goos == "linux" {
			return "aarch64"
		}
		return "arm64"
	case "386":
		return "i386"
	default:
		return goarch
	}
}

// downloadFile 下载文件
func downloadFile(url, filepath string) error {
	fmt.Print("⏬ 下载中...")

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 显示下载进度
	counter := &writeCounter{Total: resp.ContentLength}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		return err
	}

	fmt.Print(" 完成\n")
	return nil
}

// writeCounter 下载进度计数器
type writeCounter struct {
	Total      int64
	Downloaded int64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Downloaded += int64(n)
	wc.printProgress()
	return n, nil
}

func (wc *writeCounter) printProgress() {
	if wc.Total <= 0 {
		return
	}
	percent := float64(wc.Downloaded) / float64(wc.Total) * 100
	fmt.Printf("\r⏬ 下载中... %.0f%%", percent)
}

// extractTarGz 解压 tar.gz 文件
func extractTarGz(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 只解压 opsxcli 二进制文件
		if !strings.HasSuffix(header.Name, "opsxcli") && header.Name != "opsxcli" {
			continue
		}

		target := filepath.Join(destDir, "opsxcli")

		switch header.Typeflag {
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// replaceFile 替换文件
func replaceFile(src, dst string) error {
	// 读取源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	// 获取目标文件的权限
	destInfo, err := os.Stat(dst)
	if err != nil {
		return err
	}

	// 创建临时文件
	tempFile := dst + ".tmp"
	destFile, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, destInfo.Mode())
	if err != nil {
		return err
	}

	// 复制内容
	if _, err = io.Copy(destFile, sourceFile); err != nil {
		destFile.Close()
		os.Remove(tempFile)
		return err
	}
	destFile.Close()

	// 设置可执行权限
	if err := os.Chmod(tempFile, sourceInfo.Mode()); err != nil {
		os.Remove(tempFile)
		return err
	}

	// 替换原文件
	if err := os.Rename(tempFile, dst); err != nil {
		os.Remove(tempFile)
		return err
	}

	return nil
}

// cleanOldBackups 清理旧备份，只保留最近的 maxBackups 个
func cleanOldBackups(binaryPath string) error {
	// 获取二进制文件所在目录
	dir := filepath.Dir(binaryPath)
	baseName := filepath.Base(binaryPath)

	// 查找所有备份文件
	pattern := filepath.Join(dir, baseName+".backup.*")
	backups, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("查找备份文件失败: %w", err)
	}

	// 如果备份数量不超过限制，无需清理
	if len(backups) <= maxBackups {
		return nil
	}

	// 按文件修改时间排序（从新到旧）
	sort.Slice(backups, func(i, j int) bool {
		infoI, errI := os.Stat(backups[i])
		infoJ, errJ := os.Stat(backups[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// 删除超出限制的旧备份（静默处理）
	for i := maxBackups; i < len(backups); i++ {
		os.Remove(backups[i]) // 忽略错误，静默删除
	}

	return nil
}
