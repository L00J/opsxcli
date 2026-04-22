package alert

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

// Checker 定义系统指标检查接口。
type Checker interface {
	// Check 执行指定指标的检查，返回当前值。
	Check(metric string) (float64, error)
	// AvailableMetrics 返回此检查器支持的指标列表。
	AvailableMetrics() []string
}

// SystemChecker 基于 gopsutil 的系统指标检查器。
type SystemChecker struct{}

// NewSystemChecker 创建系统指标检查器。
func NewSystemChecker() *SystemChecker {
	return &SystemChecker{}
}

// Check 获取指定指标的当前值。
func (sc *SystemChecker) Check(metric string) (float64, error) {
	switch metric {
	case "cpu_percent":
		return sc.checkCPU()
	case "memory_percent":
		return sc.checkMemory()
	case "disk_percent":
		return sc.checkDisk()
	case "disk_used_gb":
		return sc.checkDiskUsedGB()
	case "load1":
		return sc.checkLoad(1)
	case "load5":
		return sc.checkLoad(5)
	case "load15":
		return sc.checkLoad(15)
	case "process_count":
		return sc.checkProcessCount()
	case "uptime_seconds":
		return sc.checkUptime()
	default:
		return 0, fmt.Errorf("不支持的指标: %q", metric)
	}
}

// AvailableMetrics 返回所有可用指标。
func (sc *SystemChecker) AvailableMetrics() []string {
	return []string{
		"cpu_percent",
		"memory_percent",
		"disk_percent",
		"disk_used_gb",
		"load1",
		"load5",
		"load15",
		"process_count",
		"uptime_seconds",
	}
}

// CheckRule 对指定规则执行检查，返回检查结果。
func CheckRule(checker Checker, rule *Rule) (*CheckResult, error) {
	metric, operator, threshold, err := ParseCheckExpr(rule.Check)
	if err != nil {
		return nil, fmt.Errorf("解析检查表达式失败: %w", err)
	}

	value, err := checker.Check(metric)
	if err != nil {
		return nil, fmt.Errorf("检查指标 %q 失败: %w", metric, err)
	}

	pass := !Evaluate(value, operator, threshold)

	var status string
	if pass {
		status = "正常"
	} else {
		status = "异常"
	}

	message := fmt.Sprintf("[%s] %s: %.1f %s %.1f (%s)",
		status, rule.Name, value, operator, threshold, rule.Description)

	return &CheckResult{
		RuleName:  rule.Name,
		Pass:      pass,
		Value:     value,
		Threshold: threshold,
		Message:   message,
		CheckedAt: time.Now(),
	}, nil
}

// CheckAllRules 对所有启用的规则执行检查。
func CheckAllRules(checker Checker, rules []Rule) ([]*CheckResult, []error) {
	enabled := FilterRules(rules)
	results := make([]*CheckResult, 0, len(enabled))
	var errs []error

	for i := range enabled {
		result, err := CheckRule(checker, &enabled[i])
		if err != nil {
			errs = append(errs, fmt.Errorf("规则 %q 检查失败: %w", enabled[i].Name, err))
			continue
		}
		results = append(results, result)
	}

	return results, errs
}

// --- 内部实现 ---

func (sc *SystemChecker) checkCPU() (float64, error) {
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return 0, fmt.Errorf("获取 CPU 使用率失败: %w", err)
	}
	if len(percentages) == 0 {
		return 0, fmt.Errorf("无法获取 CPU 使用率")
	}
	return percentages[0], nil
}

func (sc *SystemChecker) checkMemory() (float64, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("获取内存信息失败: %w", err)
	}
	return vmStat.UsedPercent, nil
}

func (sc *SystemChecker) checkDisk() (float64, error) {
	diskStat, err := disk.Usage("/")
	if err != nil {
		return 0, fmt.Errorf("获取磁盘信息失败: %w", err)
	}
	return diskStat.UsedPercent, nil
}

func (sc *SystemChecker) checkDiskUsedGB() (float64, error) {
	diskStat, err := disk.Usage("/")
	if err != nil {
		return 0, fmt.Errorf("获取磁盘信息失败: %w", err)
	}
	return float64(diskStat.Used) / 1024 / 1024 / 1024, nil
}

func (sc *SystemChecker) checkLoad(which int) (float64, error) {
	loadStat, err := load.Avg()
	if err != nil {
		return 0, fmt.Errorf("获取负载信息失败: %w", err)
	}
	switch which {
	case 1:
		return loadStat.Load1, nil
	case 5:
		return loadStat.Load5, nil
	case 15:
		return loadStat.Load15, nil
	default:
		return 0, fmt.Errorf("无效的负载指标: %d", which)
	}
}

func (sc *SystemChecker) checkProcessCount() (float64, error) {
	pids, err := cpu.Percent(0, false)
	_ = pids
	// 使用 gopsutil 的 process 包替代
	count, err := getProcessCount()
	if err != nil {
		return 0, fmt.Errorf("获取进程数失败: %w", err)
	}
	return float64(count), nil
}

func (sc *SystemChecker) checkUptime() (float64, error) {
	// 通过 /proc/uptime 或系统调用获取
	uptime, err := getUptimeSeconds()
	if err != nil {
		return 0, fmt.Errorf("获取系统运行时间失败: %w", err)
	}
	return uptime, nil
}
