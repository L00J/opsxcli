package busybox

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// ==================== free 命令 ====================

// FreeOptions free 命令选项
type FreeOptions struct {
	Human bool   // -h 人类可读
	Unit  string // -b/-k/-m/-g 单位
	Total bool   // --total 显示总计行
}

// Free 显示内存使用情况
func Free(opts FreeOptions) error {
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("获取内存信息失败: %w", err)
	}

	s, err := mem.SwapMemory()
	if err != nil {
		return fmt.Errorf("获取交换分区信息失败: %w", err)
	}

	// 格式化函数
	formatVal := formatMemoryValue(v.Total, opts)

	// 表头
	header := fmt.Sprintf("%-14s %12s %12s %12s %12s %12s", "", "total", "used", "free", "shared", "available")

	// Mem 行
	shared := v.Shared // 共享内存
	fmt.Println(header)
	fmt.Printf("%-14s %12s %12s %12s %12s %12s\n",
		"Mem:",
		formatVal(v.Total),
		formatVal(v.Used),
		formatVal(v.Free),
		formatVal(shared),
		formatVal(v.Available),
	)

	// Swap 行
	fmt.Printf("%-14s %12s %12s %12s\n",
		"Swap:",
		formatVal(s.Total),
		formatVal(s.Used),
		formatVal(s.Free),
	)

	// Total 行
	if opts.Total {
		totalTotal := v.Total + s.Total
		totalUsed := v.Used + s.Used
		totalFree := v.Free + s.Free
		fmt.Printf("%-14s %12s %12s %12s\n",
			"Total:",
			formatVal(totalTotal),
			formatVal(totalUsed),
			formatVal(totalFree),
		)
	}

	return nil
}

// formatMemoryValue 根据选项返回格式化函数
func formatMemoryValue(fallback uint64, opts FreeOptions) func(uint64) string {
	if opts.Human {
		return humanSize
	}

	unit := strings.ToLower(opts.Unit)
	switch unit {
	case "b":
		return func(v uint64) string { return strconv.FormatUint(v, 10) }
	case "k":
		return func(v uint64) string { return strconv.FormatUint(v/1024, 10) }
	case "m":
		return func(v uint64) string { return strconv.FormatUint(v/1024/1024, 10) }
	case "g":
		return func(v uint64) string { return strconv.FormatUint(v/1024/1024/1024, 10) }
	default:
		// 默认 KB（兼容 GNU free）
		return func(v uint64) string { return strconv.FormatUint(v/1024, 10) }
	}
}

// humanSize 人类可读的大小
func humanSize(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1fTi", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1fGi", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1fMi", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1fKi", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// ==================== df 命令 ====================

// DfOptions df 命令选项
type DfOptions struct {
	Human     bool   // -h 人类可读
	All       bool   // -a 显示所有文件系统
	FsType    bool   // -T 显示文件系统类型
	FilterType string // -t type 按文件系统类型过滤
}

// Df 显示磁盘空间使用情况
func Df(opts DfOptions) error {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return fmt.Errorf("获取分区信息失败: %w", err)
	}

	// 表头
	if opts.FsType {
		fmt.Printf("%-20s %-8s %10s %10s %10s %-6s %s\n",
			"Filesystem", "Type", "Size", "Used", "Avail", "Use%", "Mounted on")
	} else {
		fmt.Printf("%-20s %10s %10s %10s %-6s %s\n",
			"Filesystem", "Size", "Used", "Avail", "Use%", "Mounted on")
	}

	// 收集并排序
	type partInfo struct {
		Device     string
		Mountpoint string
		Fstype     string
		Total      uint64
		Used       uint64
		Free       uint64
		UsePercent float64
	}

	var parts []partInfo
	for _, p := range partitions {
		// 过滤虚拟文件系统
		if !opts.All && isVirtualFS(p.Fstype) {
			continue
		}

		// 类型过滤
		if opts.FilterType != "" && !strings.EqualFold(p.Fstype, opts.FilterType) {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage == nil {
			continue
		}

		// 跳过 0 大小的文件系统（除非 -a）
		if !opts.All && usage.Total == 0 {
			continue
		}

		parts = append(parts, partInfo{
			Device:     p.Device,
			Mountpoint: p.Mountpoint,
			Fstype:     p.Fstype,
			Total:      usage.Total,
			Used:       usage.Used,
			Free:       usage.Free,
			UsePercent: usage.UsedPercent,
		})
	}

	// 按挂载点排序
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Mountpoint < parts[j].Mountpoint
	})

	// 格式化
	formatSize := func(v uint64) string {
		if opts.Human {
			return humanSize(v)
		}
		return strconv.FormatUint(v/1024, 10)
	}

	for _, p := range parts {
		useStr := fmt.Sprintf("%.0f%%", p.UsePercent)
		if p.Total == 0 {
			useStr = "-"
		}

		if opts.FsType {
			fmt.Printf("%-20s %-8s %10s %10s %10s %-6s %s\n",
				p.Device, p.Fstype,
				formatSize(p.Total), formatSize(p.Used), formatSize(p.Free),
				useStr, p.Mountpoint)
		} else {
			fmt.Printf("%-20s %10s %10s %10s %-6s %s\n",
				p.Device,
				formatSize(p.Total), formatSize(p.Used), formatSize(p.Free),
				useStr, p.Mountpoint)
		}
	}

	return nil
}

// isVirtualFS 判断是否是虚拟文件系统
func isVirtualFS(fstype string) bool {
	virtual := map[string]bool{
		"sysfs": true, "proc": true, "devtmpfs": true, "devfs": true,
		"tmpfs": true, "securityfs": true, "debugfs": true,
		"cgroup": true, "cgroup2": true, "pstore": true,
		"efivarfs": true, "bpf": true, "configfs": true,
		"fusectl": true, "tracefs": true, "mqueue": true,
		"hugetlbfs": true, "rpc_pipefs": true, "overlay": true,
		"autofs": true, "binfmt_misc": true, "fuse.gvfsd-fuse": true,
	}
	return virtual[fstype]
}

// ==================== kill 命令 ====================

// KillOptions kill 命令选项
type KillOptions struct {
	Signal    string // -s signal
	List      bool   // -l 列出信号
}

// 信号映射表
var signalMap = map[string]syscall.Signal{
	"TERM":  syscall.SIGTERM,
	"HUP":   syscall.SIGHUP,
	"INT":   syscall.SIGINT,
	"KILL":  syscall.SIGKILL,
	"USR1":  syscall.SIGUSR1,
	"USR2":  syscall.SIGUSR2,
	"STOP":  syscall.SIGSTOP,
	"CONT":  syscall.SIGCONT,
	"QUIT":  syscall.SIGQUIT,
	"ALRM":  syscall.SIGALRM,
	"PIPE":  syscall.SIGPIPE,
	"ABRT":  syscall.SIGABRT,
	"TSTP":  syscall.SIGTSTP,
	"TTIN":  syscall.SIGTTIN,
	"TTOU":  syscall.SIGTTOU,
	"SEGV":  syscall.SIGSEGV,
	"CHLD":  syscall.SIGCHLD,
}

// Kill 发送信号给进程
func Kill(args []string, opts KillOptions) error {
	// -l 列出所有信号
	if opts.List {
		listSignals()
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("kill: 需要指定 PID")
	}

	// 解析信号
	sig := syscall.SIGTERM // 默认 SIGTERM
	if opts.Signal != "" {
		var err error
		sig, err = parseSignal(opts.Signal)
		if err != nil {
			return err
		}
	}

	// 发送信号给每个 PID
	hasError := false
	for _, arg := range args {
		pid, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: 无效的 PID: %s\n", arg)
			hasError = true
			continue
		}

		if err := syscall.Kill(pid, sig); err != nil {
			fmt.Fprintf(os.Stderr, "kill: (%d) - %v\n", pid, err)
			hasError = true
		}
	}

	if hasError {
		return fmt.Errorf("")
	}
	return nil
}

// parseSignal 解析信号名或编号
func parseSignal(s string) (syscall.Signal, error) {
	// 去掉 SIG 前缀
	name := strings.TrimPrefix(strings.ToUpper(s), "SIG")

	// 尝试按名称解析
	if sig, ok := signalMap[name]; ok {
		return sig, nil
	}

	// 尝试按编号解析
	num, err := strconv.Atoi(name)
	if err != nil {
		return 0, fmt.Errorf("kill: 未知信号: %s", s)
	}

	// 查找编号对应的信号
	for sigName, sig := range signalMap {
		if int(sig) == num {
			_ = sigName
			return sig, nil
		}
	}

	return syscall.Signal(num), nil
}

// listSignals 列出所有支持的信号
func listSignals() {
	// 按编号排序
	type sigEntry struct {
		name   string
		number int
	}
	var entries []sigEntry
	seen := map[int]bool{}
	for name, sig := range signalMap {
		n := int(sig)
		if !seen[n] {
			seen[n] = true
			entries = append(entries, sigEntry{name, n})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].number < entries[j].number
	})

	// 格式化输出
	col := 0
	for _, e := range entries {
		fmt.Printf("%2d) %-8s", e.number, e.name)
		col++
		if col%6 == 0 {
			fmt.Println()
		}
	}
	if col%6 != 0 {
		fmt.Println()
	}
}

