package builtin

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LsOptions ls 命令选项
type LsOptions struct {
	All        bool // -a 显示隐藏文件
	Long       bool // -l 详细列表
	Human      bool // -h 人类可读的大小
	Recursive  bool // -R 递归显示
	SortByTime bool // -t 按时间排序
	Reverse    bool // -r 反向排序
}

// Ls 实现 ls 命令
func Ls(paths []string, opts LsOptions) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, path := range paths {
		if err := listPath(path, opts); err != nil {
			fmt.Fprintf(os.Stderr, "ls: %s: %v\n", path, err)
		}
	}

	return nil
}

func listPath(path string, opts LsOptions) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		printFileInfo(path, info, opts)
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// 获取文件信息
	var files []os.FileInfo
	for _, entry := range entries {
		if !opts.All && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, info)
	}

	// 排序
	if opts.SortByTime {
		sort.Slice(files, func(i, j int) bool {
			if opts.Reverse {
				return files[i].ModTime().Before(files[j].ModTime())
			}
			return files[i].ModTime().After(files[j].ModTime())
		})
	} else {
		sort.Slice(files, func(i, j int) bool {
			if opts.Reverse {
				return files[i].Name() > files[j].Name()
			}
			return files[i].Name() < files[j].Name()
		})
	}

	// 打印
	for _, file := range files {
		fullPath := filepath.Join(path, file.Name())
		printFileInfo(fullPath, file, opts)
	}

	return nil
}

func printFileInfo(path string, info os.FileInfo, opts LsOptions) {
	if opts.Long {
		// 详细格式
		mode := info.Mode()
		size := info.Size()
		modTime := info.ModTime().Format("Jan 02 15:04")

		sizeStr := fmt.Sprintf("%d", size)
		if opts.Human {
			sizeStr = formatSize(size)
		}

		fmt.Printf("%s %4d %8s %12s %s %s\n",
			mode.String(), 1, "root", sizeStr, modTime, filepath.Base(path))
	} else {
		// 简单格式
		fmt.Println(filepath.Base(path))
	}
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%dB", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(size)/float64(div), "KMGTPE"[exp])
}

// Cat 实现 cat 命令
func Cat(files []string, showLineNumbers bool) error {
	if len(files) == 0 {
		// 从标准输入读取
		catReader(os.Stdin, showLineNumbers, 1)
		return nil
	}

	lineNum := 1
	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cat: %s: %v\n", filename, err)
			continue
		}

		lineNum = catReader(file, showLineNumbers, lineNum)
		file.Close()
	}

	return nil
}

func catReader(r io.Reader, showLineNumbers bool, startLine int) int {
	scanner := bufio.NewScanner(r)
	lineNum := startLine

	for scanner.Scan() {
		if showLineNumbers {
			fmt.Printf("%6d  %s\n", lineNum, scanner.Text())
			lineNum++
		} else {
			fmt.Println(scanner.Text())
		}
	}

	return lineNum
}

// Grep 实现 grep 命令
func Grep(pattern string, files []string, opts GrepOptions) error {
	if len(files) == 0 {
		return grepReader(os.Stdin, "<stdin>", pattern, opts)
	}

	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "grep: %s: %v\n", filename, err)
			continue
		}

		grepReader(file, filename, pattern, opts)
		file.Close()
	}

	return nil
}

type GrepOptions struct {
	IgnoreCase   bool // -i 忽略大小写
	InvertMatch  bool // -v 反向匹配
	LineNumber   bool // -n 显示行号
	CountOnly    bool // -c 只显示计数
	WithFilename bool // -H 显示文件名
}

func grepReader(r io.Reader, filename string, pattern string, opts GrepOptions) error {
	scanner := bufio.NewScanner(r)
	lineNum := 0
	matchCount := 0

	searchPattern := pattern
	if opts.IgnoreCase {
		searchPattern = strings.ToLower(pattern)
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		searchLine := line
		if opts.IgnoreCase {
			searchLine = strings.ToLower(line)
		}

		matched := strings.Contains(searchLine, searchPattern)
		if opts.InvertMatch {
			matched = !matched
		}

		if matched {
			matchCount++
			if !opts.CountOnly {
				prefix := ""
				if opts.WithFilename {
					prefix = filename + ":"
				}
				if opts.LineNumber {
					prefix += fmt.Sprintf("%d:", lineNum)
				}
				fmt.Printf("%s%s\n", prefix, line)
			}
		}
	}

	if opts.CountOnly {
		if opts.WithFilename {
			fmt.Printf("%s:", filename)
		}
		fmt.Println(matchCount)
	}

	return scanner.Err()
}

// Cp 实现 cp 命令
func Cp(src, dst string, recursive bool) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if srcInfo.IsDir() {
		if !recursive {
			return fmt.Errorf("cp: -r not specified; omitting directory '%s'", src)
		}
		return copyDir(src, dst)
	}

	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// 复制权限
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Mv 实现 mv 命令
func Mv(src, dst string) error {
	// 先尝试直接重命名（同文件系统）
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// 如果失败，则复制后删除（跨文件系统）
	if err := copyFile(src, dst); err != nil {
		return err
	}

	return os.Remove(src)
}

// Rm 实现 rm 命令
func Rm(paths []string, recursive, force bool) error {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			if !force {
				fmt.Fprintf(os.Stderr, "rm: %s: %v\n", path, err)
			}
			continue
		}

		if info.IsDir() {
			if !recursive {
				if !force {
					fmt.Fprintf(os.Stderr, "rm: %s: is a directory\n", path)
				}
				continue
			}
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}

		if err != nil && !force {
			fmt.Fprintf(os.Stderr, "rm: %s: %v\n", path, err)
		}
	}

	return nil
}

// Mkdir 实现 mkdir 命令
func Mkdir(paths []string, parents bool, mode os.FileMode) error {
	for _, path := range paths {
		var err error
		if parents {
			err = os.MkdirAll(path, mode)
		} else {
			err = os.Mkdir(path, mode)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "mkdir: %s: %v\n", path, err)
		}
	}

	return nil
}

// Touch 实现 touch 命令
func Touch(paths []string) error {
	now := time.Now()

	for _, path := range paths {
		// 如果文件不存在，创建它
		if _, err := os.Stat(path); os.IsNotExist(err) {
			file, err := os.Create(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "touch: %s: %v\n", path, err)
				continue
			}
			file.Close()
		} else {
			// 更新时间戳
			if err := os.Chtimes(path, now, now); err != nil {
				fmt.Fprintf(os.Stderr, "touch: %s: %v\n", path, err)
			}
		}
	}

	return nil
}

// Rmdir 实现 rmdir 命令
func Rmdir(paths []string, parents bool) error {
	for _, path := range paths {
		if parents {
			// -p 选项: 递归删除空目录(从最深层开始)
			err := rmdirParents(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "rmdir: %s: %v\n", path, err)
			}
		} else {
			// 只删除空目录
			err := os.Remove(path)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "rmdir: %s: 目录不存在\n", path)
				} else {
					fmt.Fprintf(os.Stderr, "rmdir: %s: %v\n", path, err)
				}
			}
		}
	}
	return nil
}

func rmdirParents(path string) error {
	// 从给定路径开始,逐级向上删除空目录
	for path != "" && path != "/" && path != "." {
		err := os.Remove(path)
		if err != nil {
			// 如果目录不为空或不存在,停止删除
			return err
		}
		// 移动到父目录
		path = filepath.Dir(path)
		if path == "/" || path == "." {
			break
		}
	}
	return nil
}

// Chmod 实现 chmod 命令
func Chmod(mode string, paths []string, recursive bool) error {
	// 解析权限模式 (例如: 755, 0644)
	var fileMode os.FileMode
	_, err := fmt.Sscanf(mode, "%o", &fileMode)
	if err != nil {
		return fmt.Errorf("无效的权限模式: %s", mode)
	}

	for _, path := range paths {
		if recursive {
			err := chmodRecursive(path, fileMode)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chmod: %s: %v\n", path, err)
			}
		} else {
			err := os.Chmod(path, fileMode)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chmod: %s: %v\n", path, err)
			}
		}
	}
	return nil
}

func chmodRecursive(path string, mode os.FileMode) error {
	err := os.Chmod(path, mode)
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			if err := chmodRecursive(entryPath, mode); err != nil {
				return err
			}
		}
	}

	return nil
}

// Chown 实现 chown 命令
func Chown(owner string, paths []string, recursive bool) error {
	// 解析 owner:group 格式
	// 注意: 这是简化实现,完整版需要查询用户/组 ID
	// 在容器环境中通常直接使用 UID:GID
	var uid, gid int
	parts := strings.Split(owner, ":")
	if len(parts) == 2 {
		fmt.Sscanf(parts[0], "%d", &uid)
		fmt.Sscanf(parts[1], "%d", &gid)
	} else {
		fmt.Sscanf(parts[0], "%d", &uid)
		gid = -1 // 不改变组
	}

	for _, path := range paths {
		if recursive {
			err := chownRecursive(path, uid, gid)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chown: %s: %v\n", path, err)
			}
		} else {
			err := os.Chown(path, uid, gid)
			if err != nil {
				fmt.Fprintf(os.Stderr, "chown: %s: %v\n", path, err)
			}
		}
	}
	return nil
}

func chownRecursive(path string, uid, gid int) error {
	err := os.Chown(path, uid, gid)
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			entryPath := filepath.Join(path, entry.Name())
			if err := chownRecursive(entryPath, uid, gid); err != nil {
				return err
			}
		}
	}

	return nil
}

// Ln 实现 ln 命令
func Ln(target, link string, symbolic bool) error {
	var err error
	if symbolic {
		// 创建符号链接
		err = os.Symlink(target, link)
	} else {
		// 创建硬链接
		err = os.Link(target, link)
	}

	if err != nil {
		return fmt.Errorf("ln: %v", err)
	}

	return nil
}

// Head 实现 head 命令
func Head(files []string, lines int) error {
	if lines <= 0 {
		lines = 10 // 默认显示 10 行
	}

	if len(files) == 0 {
		// 从标准输入读取
		return headReader(os.Stdin, "<stdin>", lines, false)
	}

	showFilename := len(files) > 1

	for i, filename := range files {
		if showFilename && i > 0 {
			fmt.Println() // 文件之间空行分隔
		}

		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "head: %s: %v\n", filename, err)
			continue
		}

		if showFilename {
			fmt.Printf("==> %s <==\n", filename)
		}

		headReader(file, filename, lines, showFilename)
		file.Close()
	}

	return nil
}

func headReader(r io.Reader, filename string, lines int, showFilename bool) error {
	scanner := bufio.NewScanner(r)
	count := 0

	for scanner.Scan() && count < lines {
		fmt.Println(scanner.Text())
		count++
	}

	return scanner.Err()
}

// Tail 实现 tail 命令
func Tail(files []string, lines int) error {
	if lines <= 0 {
		lines = 10 // 默认显示 10 行
	}

	if len(files) == 0 {
		// 从标准输入读取
		return tailReader(os.Stdin, "<stdin>", lines, false)
	}

	showFilename := len(files) > 1

	for i, filename := range files {
		if showFilename && i > 0 {
			fmt.Println() // 文件之间空行分隔
		}

		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tail: %s: %v\n", filename, err)
			continue
		}

		if showFilename {
			fmt.Printf("==> %s <==\n", filename)
		}

		tailReader(file, filename, lines, showFilename)
		file.Close()
	}

	return nil
}

func tailReader(r io.Reader, filename string, lines int, showFilename bool) error {
	scanner := bufio.NewScanner(r)

	// 使用环形缓冲区保存最后 N 行
	buffer := make([]string, lines)
	count := 0
	idx := 0

	for scanner.Scan() {
		buffer[idx] = scanner.Text()
		idx = (idx + 1) % lines
		count++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 输出缓冲区中的内容
	start := 0
	if count > lines {
		start = idx
	}

	displayCount := count
	if displayCount > lines {
		displayCount = lines
	}

	for i := 0; i < displayCount; i++ {
		fmt.Println(buffer[(start+i)%lines])
	}

	return nil
}
