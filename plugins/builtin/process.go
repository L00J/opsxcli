package builtin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// ==================== ps 命令 ====================

// PsOptions ps 命令选项
type PsOptions struct {
	Full     bool // -f 完整格式
	AllUsers bool // -a 所有用户
	ShowAll  bool // -A/-e 所有进程
	Human    bool // -h 人类可读内存
}

// Ps 显示进程列表
func Ps(opts PsOptions) error {
	pids, err := process.Processes()
	if err != nil {
		return fmt.Errorf("获取进程列表失败: %w", err)
	}

	// 表头
	if opts.Full {
		fmt.Printf("%-8s %-8s %-8s %-6s %-10s %s\n",
			"UID", "PID", "PPID", "CPU%", "MEM%", "COMMAND")
	} else {
		fmt.Printf("%-8s %-6s %-10s %s\n", "PID", "CPU%", "MEM%", "COMMAND")
	}

	type procInfo struct {
		UID    string
		PID    int32
		PPID   int32
		CPU    float64
		Mem    float32
		Cmd    string
	}

	var procs []procInfo
	for _, p := range pids {
		// 获取进程名
		name, _ := p.Name()
		cmdline, _ := p.Cmdline()
		if cmdline == "" {
			cmdline = name
		}
		// 截断过长的命令行
		if len(cmdline) > 80 {
			cmdline = cmdline[:77] + "..."
		}

		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()

		uid := "-"
		if uids, err := p.Uids(); err == nil && len(uids) > 0 {
			uid = strconv.Itoa(int(uids[0]))
		}

		ppid := int32(0)
		if pp, err := p.Ppid(); err == nil {
			ppid = pp
		}

		procs = append(procs, procInfo{
			UID:  uid,
			PID:  p.Pid,
			PPID: ppid,
			CPU:  cpu,
			Mem:  mem,
			Cmd:  cmdline,
		})
	}

	// 按 PID 排序
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].PID < procs[j].PID
	})

	for _, p := range procs {
		if opts.Full {
			fmt.Printf("%-8s %-8d %-8d %-6.1f %-10.1f %s\n",
				p.UID, p.PID, p.PPID, p.CPU, p.Mem, p.Cmd)
		} else {
			fmt.Printf("%-8d %-6.1f %-10.1f %s\n",
				p.PID, p.CPU, p.Mem, p.Cmd)
		}
	}

	return nil
}

// ==================== pstree 命令 ====================

// PstreeOptions pstree 命令选项
type PstreeOptions struct {
	PID      int  // -p 指定根进程 PID
	ShowPID  bool // -p 显示 PID
	FullCmd  bool // -a 显示完整命令行
}

// Pstree 以树形结构显示进程
func Pstree(opts PstreeOptions) error {
	pids, err := process.Processes()
	if err != nil {
		return fmt.Errorf("获取进程列表失败: %w", err)
	}

	// 构建 PPID → []PID 映射
	children := make(map[int32][]int32)
	procMap := make(map[int32]*process.Process)
	for _, p := range pids {
		ppid, _ := p.Ppid()
		children[ppid] = append(children[ppid], p.Pid)
		procMap[p.Pid] = p
	}

	// 排序子进程
	for ppid := range children {
		sort.Slice(children[ppid], func(i, j int) bool {
			return children[ppid][i] < children[ppid][j]
		})
	}

	// 确定根进程
	rootPID := int32(1) // 默认从 PID 1 开始
	if opts.PID > 0 {
		rootPID = int32(opts.PID)
	}

	// 如果不指定 PID，找到所有根进程（PPID=0 或不在进程列表中的）
	if opts.PID <= 0 {
		// 查找 init 进程或所有根进程
		if p, exists := procMap[1]; exists {
			name, _ := p.Name()
			printProcessTree(p, 1, procMap, children, "", true, opts)
			_ = name
		} else {
			// 找所有根进程
			for _, p := range pids {
				ppid, _ := p.Ppid()
				if _, exists := procMap[ppid]; !exists || ppid == 0 {
					printProcessTree(p, p.Pid, procMap, children, "", true, opts)
				}
			}
		}
	} else {
		if p, exists := procMap[rootPID]; exists {
			printProcessTree(p, rootPID, procMap, children, "", true, opts)
		} else {
			return fmt.Errorf("pstree: 进程 %d 不存在", rootPID)
		}
	}

	return nil
}

func printProcessTree(p *process.Process, pid int32, procMap map[int32]*process.Process, children map[int32][]int32, prefix string, isLast bool, opts PstreeOptions) {
	name, _ := p.Name()

	if opts.FullCmd {
		if cmdline, err := p.Cmdline(); err == nil && cmdline != "" {
			name = cmdline
			if len(name) > 60 {
				name = name[:57] + "..."
			}
		}
	}

	display := name
	if opts.ShowPID {
		display = fmt.Sprintf("%s(%d)", name, pid)
	}

	if prefix == "" {
		// 根进程
		fmt.Println(display)
	} else {
		connector := "├── "
		if isLast {
			connector = "└── "
		}
		fmt.Printf("%s%s%s\n", prefix, connector, display)
	}

	// 递归子进程
	subs, exists := children[pid]
	if !exists {
		return
	}

	newPrefix := prefix
	if prefix != "" {
		if isLast {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
	}

	for i, childPID := range subs {
		childLast := i == len(subs)-1
		if childProc, ok := procMap[childPID]; ok {
			printProcessTree(childProc, childPID, procMap, children, newPrefix, childLast, opts)
		}
	}
}

// ==================== top 命令 ====================

// TopOptions top 命令选项
type TopOptions struct {
	Delay    int  // -d 刷新间隔(秒)
	Count    int  // -n 刷新次数
	ShowAll  bool // -a 显示全部进程
}

// Top 显示系统进程（简化版，指向 opsxcli sys）
func Top(opts TopOptions) error {
	// 简化版：直接显示一次进程列表
	fmt.Printf("%-8s %-8s %-6s %-10s %-10s %s\n",
		"PID", "USER", "CPU%", "MEM%", "TIME+", "COMMAND")
	fmt.Println(strings.Repeat("-", 75))

	pids, err := process.Processes()
	if err != nil {
		return fmt.Errorf("获取进程列表失败: %w", err)
	}

	type procInfo struct {
		PID   int32
		User  string
		CPU   float64
		Mem   float32
		Cmd   string
	}

	var procs []procInfo
	for _, p := range pids {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()

		user := "-"
		if uids, err := p.Uids(); err == nil && len(uids) > 0 {
			user = strconv.Itoa(int(uids[0]))
		}

		procs = append(procs, procInfo{
			PID:  p.Pid,
			User: user,
			CPU:  cpu,
			Mem:  mem,
			Cmd:  name,
		})
	}

	// 按 CPU 降序
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})

	limit := 20
	if opts.ShowAll || len(procs) < limit {
		limit = len(procs)
	}

	for i := 0; i < limit; i++ {
		p := procs[i]
		fmt.Printf("%-8d %-8s %-6.1f %-10.1f %-10s %s\n",
			p.PID, p.User, p.CPU, p.Mem, "-", p.Cmd)
	}

	fmt.Printf("\n显示 %d/%d 个进程 (按 CPU 排序，使用 opsxcli sys 查看完整监控)\n", limit, len(procs))
	return nil
}
