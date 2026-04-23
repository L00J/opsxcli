package builtin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ==================== gzip 命令 ====================

// GzipOptions gzip 命令选项
type GzipOptions struct {
	Decompress bool // -d 解压
	Keep       bool // -k 保留原文件
	Stdout     bool // -c 输出到 stdout
}

// Gzip 压缩或解压文件
func Gzip(args []string, opts GzipOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("gzip: 需要指定文件")
	}

	hasError := false
	for _, filename := range args {
		if opts.Decompress {
			if err := gunzipFile(filename, opts); err != nil {
				fmt.Fprintf(os.Stderr, "gzip: %v\n", err)
				hasError = true
			}
		} else {
			if err := gzipFile(filename, opts); err != nil {
				fmt.Fprintf(os.Stderr, "gzip: %v\n", err)
				hasError = true
			}
		}
	}

	if hasError {
		return fmt.Errorf("gzip: 部分文件处理失败")
	}
	return nil
}

func gzipFile(filename string, opts GzipOptions) error {
	src, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	outName := filename + ".gz"
	var w io.Writer
	if opts.Stdout {
		w = os.Stdout
	} else {
		f, err := os.Create(outName)
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}
		defer f.Close()
		w = f
	}

	gw := gzip.NewWriter(w)
	defer gw.Close()

	if _, err := io.Copy(gw, src); err != nil {
		return fmt.Errorf("压缩失败: %w", err)
	}

	if !opts.Stdout && !opts.Keep {
		os.Remove(filename)
	}
	return nil
}

func gunzipFile(filename string, opts GzipOptions) error {
	src, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	gr, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	defer gr.Close()

	outName := strings.TrimSuffix(filename, ".gz")
	if outName == filename {
		return fmt.Errorf("文件名不以 .gz 结尾: %s", filename)
	}

	var w io.Writer
	if opts.Stdout {
		w = os.Stdout
	} else {
		f, err := os.Create(outName)
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}
		defer f.Close()
		w = f
	}

	if _, err := io.Copy(w, gr); err != nil {
		return fmt.Errorf("解压写入失败: %w", err)
	}

	if !opts.Stdout && !opts.Keep {
		os.Remove(filename)
	}
	return nil
}

// ==================== unzip 命令 ====================

// UnzipOptions unzip 命令选项
type UnzipOptions struct {
	List  bool   // -l 列出内容（不解压）
	Dir   string // -d 目标目录
	Quiet bool   // -q 静默模式
}

// Unzip 解压 ZIP 文件
func Unzip(args []string, opts UnzipOptions) error {
	if len(args) == 0 {
		return fmt.Errorf("unzip: 需要指定 ZIP 文件")
	}

	zipfile := args[0]
	r, err := zip.OpenReader(zipfile)
	if err != nil {
		return fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer r.Close()

	// 列出模式
	if opts.List {
		fmt.Printf("  Length  Date        Time    Name\n")
		fmt.Printf("---------  ---------- -----   ----\n")
		var totalSize uint64
		for _, f := range r.File {
			modTime := f.Modified
			fmt.Printf("%9d  %s  %s\n", f.UncompressedSize64,
				modTime.Format("2006-01-02 15:04"),
				f.Name)
			totalSize += f.UncompressedSize64
		}
		fmt.Printf("---------                    -------\n")
		fmt.Printf("%9d                    %d files\n", totalSize, len(r.File))
		return nil
	}

	// 解压模式
	dest := opts.Dir
	if dest == "" {
		dest = "."
	}

	for _, f := range r.File {
		if err := extractZipFile(f, dest, opts.Quiet); err != nil {
			fmt.Fprintf(os.Stderr, "unzip: %v\n", err)
		} else if !opts.Quiet {
			fmt.Printf("  extracting: %s\n", f.Name)
		}
	}

	return nil
}

func extractZipFile(f *zip.File, dest string, quiet bool) error {
	path := filepath.Join(dest, f.Name)

	// 目录
	if f.FileInfo().IsDir() {
		return os.MkdirAll(path, f.Mode())
	}

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	w, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, rc)
	return err
}

// ==================== tar 命令 ====================

// TarOptions tar 命令选项
type TarOptions struct {
	Create    bool   // -c 创建归档
	Extract   bool   // -x 解压归档
	List      bool   // -t 列出内容
	Gzip      bool   // -z 使用 gzip
	File      string // -f 归档文件名
	Directory string // -C 切换目录
	Verbose   bool   // -v 详细输出
}

// Tar 创建或解压 tar 归档
func Tar(args []string, opts TarOptions) error {
	switch {
	case opts.Create:
		return tarCreate(args, opts)
	case opts.Extract:
		return tarExtract(opts)
	case opts.List:
		return tarList(opts)
	default:
		return fmt.Errorf("tar: 需要指定操作: -c (创建), -x (解压), -t (列出)")
	}
}

func tarCreate(files []string, opts TarOptions) error {
	if opts.File == "" {
		return fmt.Errorf("tar: 需要指定归档文件名 (-f)")
	}

	out, err := os.Create(opts.File)
	if err != nil {
		return fmt.Errorf("创建归档文件失败: %w", err)
	}
	defer out.Close()

	var w io.WriteCloser = out
	if opts.Gzip {
		gw := gzip.NewWriter(out)
		defer gw.Close()
		w = gw
	}

	tw := tar.NewWriter(w)
	defer tw.Close()

	dir := opts.Directory
	for _, file := range files {
		if err := addToTar(tw, file, dir, opts.Verbose); err != nil {
			fmt.Fprintf(os.Stderr, "tar: %v\n", err)
		}
	}

	return nil
}

func addToTar(tw *tar.Writer, path, baseDir string, verbose bool) error {
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算归档内的路径
		relPath := filePath
		if baseDir != "" {
			relPath, _ = filepath.Rel(baseDir, filePath)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if verbose {
			fmt.Println(relPath)
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}

func tarExtract(opts TarOptions) error {
	if opts.File == "" {
		return fmt.Errorf("tar: 需要指定归档文件名 (-f)")
	}

	f, err := os.Open(opts.File)
	if err != nil {
		return fmt.Errorf("打开归档文件失败: %w", err)
	}
	defer f.Close()

	var tr *tar.Reader
	if opts.Gzip {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("解压 gzip 失败: %w", err)
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(f)
	}

	dest := opts.Directory
	if dest == "" {
		dest = "."
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			w, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				fmt.Fprintf(os.Stderr, "tar: %v\n", err)
				continue
			}
			io.Copy(w, tr)
			w.Close()
		}

		if opts.Verbose {
			fmt.Println(header.Name)
		}
	}

	return nil
}

func tarList(opts TarOptions) error {
	if opts.File == "" {
		return fmt.Errorf("tar: 需要指定归档文件名 (-f)")
	}

	f, err := os.Open(opts.File)
	if err != nil {
		return fmt.Errorf("打开归档文件失败: %w", err)
	}
	defer f.Close()

	var tr *tar.Reader
	if opts.Gzip {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("解压 gzip 失败: %w", err)
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(f)
	}

	fmt.Printf("%-10s %-8s %-8s %s\n", "权限", "大小", "修改时间", "文件名")
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fmt.Printf("%-10s %8d  %s  %s\n",
			os.FileMode(header.Mode).String(),
			header.Size,
			header.ModTime.Format("2006-01-02 15:04"),
			header.Name)
	}

	return nil
}
