package builtin

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ==================== tree 命令 ====================

// TreeOptions tree 命令选项
type TreeOptions struct {
	MaxDepth   int  // -L 层级深度
	All        bool // -a 显示隐藏文件
	DirsOnly   bool // -d 只显示目录
	Fullpath   bool // -f 显示完整路径
}

// Tree 以树形结构显示目录内容
func Tree(args []string, opts TreeOptions) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("tree: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("tree: %s 不是目录", path)
	}

	fmt.Println(path)
	dirCount, fileCount := printTree(path, "", opts, 0)

	fmt.Printf("\n%d directories, %d files\n", dirCount, fileCount)
	return nil
}

func printTree(path, prefix string, opts TreeOptions, depth int) (int, int) {
	// 深度限制
	if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
		return 0, 0
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tree: %v\n", err)
		return 0, 0
	}

	// 过滤隐藏文件
	var filtered []fs.DirEntry
	for _, e := range entries {
		if !opts.All && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		filtered = append(filtered, e)
	}

	// 排序：目录在前，然后按名称
	sort.Slice(filtered, func(i, j int) bool {
		iDir := filtered[i].IsDir()
		jDir := filtered[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return filtered[i].Name() < filtered[j].Name()
	})

	var dirCount, fileCount int
	for i, entry := range filtered {
		isLast := i == len(filtered)-1

		// 构建前缀
		connector := "├── "
		newPrefix := "│   "
		if isLast {
			connector = "└── "
			newPrefix = "    "
		}

		name := entry.Name()
		displayPath := name
		if opts.Fullpath {
			displayPath = filepath.Join(path, name)
		}

		// 显示文件/目录标记
		if entry.IsDir() {
			fmt.Printf("%s%s%s/\n", prefix, connector, displayPath)
			dirCount++
			d, f := printTree(filepath.Join(path, name), prefix+newPrefix, opts, depth+1)
			dirCount += d
			fileCount += f
		} else if !opts.DirsOnly {
			fmt.Printf("%s%s%s\n", prefix, connector, displayPath)
			fileCount++
		}
	}

	return dirCount, fileCount
}

// ==================== du 命令 ====================

// DuOptions du 命令选项
type DuOptions struct {
	Human     bool // -h 人类可读
	Summarize bool // -s 只显示总计
	MaxDepth  int  // --max-depth N 深度
	All       bool // -a 显示文件大小
}

// Du 显示目录空间使用情况
func Du(args []string, opts DuOptions) error {
	if len(args) == 0 {
		args = []string{"."}
	}

	hasError := false
	for _, path := range args {
		if err := duPath(path, opts); err != nil {
			fmt.Fprintf(os.Stderr, "du: %v\n", err)
			hasError = true
		}
	}

	if hasError {
		return fmt.Errorf("")
	}
	return nil
}

func duPath(rootPath string, opts DuOptions) error {
	info, err := os.Stat(rootPath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		size := info.Size()
		printSize(size, rootPath, opts.Human)
		return nil
	}

	if opts.Summarize {
		size, err := calcDirSize(rootPath)
		if err != nil {
			return err
		}
		printSize(size, rootPath, opts.Human)
		return nil
	}

	// 按深度遍历
	return walkDu(rootPath, rootPath, opts, 0)
}

func walkDu(path, displayPath string, opts DuOptions, depth int) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	var totalSize int64
	var subdirs []string

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			size, err := calcDirSize(fullPath)
			if err == nil {
				totalSize += size
				subdirs = append(subdirs, fullPath)
				if opts.MaxDepth <= 0 || depth < opts.MaxDepth-1 {
					relPath := fullPath
					if strings.HasPrefix(relPath, "./") == false && !strings.HasPrefix(relPath, "/") {
						relPath = "./" + relPath
					}
					_ = relPath // subdirs will be walked below
				}
			}
		} else {
			info, err := entry.Info()
			if err == nil {
				totalSize += info.Size()
				if opts.All {
					printSize(info.Size(), fullPath, opts.Human)
				}
			}
		}
	}

	printSize(totalSize, displayPath, opts.Human)

	// 递归子目录
	if opts.MaxDepth <= 0 || depth < opts.MaxDepth-1 {
		for _, subdir := range subdirs {
			walkDu(subdir, subdir, opts, depth+1)
		}
	}

	return nil
}

// calcDirSize 计算目录总大小
func calcDirSize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

// printSize 打印大小和路径
func printSize(size int64, path string, human bool) {
	if human {
		fmt.Printf("%s\t%s\n", humanSize(uint64(size)), path)
	} else {
		fmt.Printf("%d\t%s\n", size/1024, path)
	}
}
