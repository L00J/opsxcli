package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"opsxcli/plugins/alert"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("alert", "监控", "告警规则引擎与定时检查", NewAlertCmd)
}

// NewAlertCmd creates the alert command.
func NewAlertCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alert <subcommand>",
		Short: "告警规则引擎",
		Long: `告警规则引擎 — 定时检查系统指标并触发通知。

支持 CPU、内存、磁盘、负载等系统指标的监控，通过飞书/钉钉/Webhook 发送告警通知。
支持告警静默、升级策略和完整的告警生命周期管理。

子命令：
  check     执行一次规则检查
  rules     管理告警规则
  history   查询告警历史
  silence   管理告警静默规则
  init      生成默认告警配置文件

示例：
  opsxcli alert check --config alert_rules.yaml     # 使用配置文件执行检查
  opsxcli alert check --cpu 80                      # 快速检查 CPU 是否超过 80%
  opsxcli alert rules --config alert_rules.yaml     # 列出所有规则
  opsxcli alert init                                # 生成默认配置文件
  opsxcli alert history --limit 20                  # 查看最近 20 条告警历史`,
	}

	cmd.AddCommand(newAlertCheckCmd())
	cmd.AddCommand(newAlertRulesCmd())
	cmd.AddCommand(newAlertHistoryCmd())
	cmd.AddCommand(newAlertSilenceCmd())
	cmd.AddCommand(newAlertInitCmd())

	return cmd
}

// newAlertCheckCmd 创建 check 子命令。
func newAlertCheckCmd() *cobra.Command {
	var (
		config string
		quick  string // 快速检查模式: "cpu > 80"
		output string // 输出格式: text, json
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "执行告警规则检查",
		Long: `执行一次告警规则检查。

可使用 --config 加载配置文件检查所有规则，也可使用快捷参数检查单个指标。

示例：
  opsxcli alert check --config alert_rules.yaml
  opsxcli alert check --cpu 80
  opsxcli alert check --memory 90 --disk 85
  opsxcli alert check --load5 4.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			checker := alert.NewSystemChecker()

			// 快速检查模式
			if quick != "" || cmd.Flags().Changed("cpu") || cmd.Flags().Changed("memory") ||
				cmd.Flags().Changed("disk") || cmd.Flags().Changed("load1") ||
				cmd.Flags().Changed("load5") || cmd.Flags().Changed("load15") {
				return runQuickCheck(cmd, checker, output)
			}

			// 配置文件模式
			if config == "" {
				// 默认配置文件路径
				homeDir, _ := os.UserHomeDir()
				config = filepath.Join(homeDir, ".opsxcli", "alert_rules.yaml")
			}

			ruleConfig, err := alert.LoadRules(config)
			if err != nil {
				return fmt.Errorf("加载规则配置失败: %w", err)
			}

			results, errs := alert.CheckAllRules(checker, ruleConfig.Rules)

			// 输出结果
			if output == "json" {
				data, _ := json.MarshalIndent(results, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
			} else {
				if len(results) == 0 && len(errs) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "没有启用的规则")
					return nil
				}

				fmt.Fprintln(cmd.OutOrStdout(), "=== 告警检查结果 ===")
				for _, r := range results {
					fmt.Fprintln(cmd.OutOrStdout(), alert.FormatCheckResult(r))
				}

				failed := 0
				for _, r := range results {
					if !r.Pass {
						failed++
					}
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n检查 %d 条规则，%d 条异常\n", len(results), failed)
			}

			// 输出错误
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "错误: %v\n", e)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&config, "config", "c", "", "告警规则配置文件路径")
	cmd.Flags().StringVar(&quick, "quick", "", "快速检查表达式 (如 'cpu_percent > 80')")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "输出格式 (text/json)")
	cmd.Flags().Float64("cpu", 0, "快速检查 CPU 使用率阈值")
	cmd.Flags().Float64("memory", 0, "快速检查内存使用率阈值")
	cmd.Flags().Float64("disk", 0, "快速检查磁盘使用率阈值")
	cmd.Flags().Float64("load1", 0, "快速检查 1 分钟负载阈值")
	cmd.Flags().Float64("load5", 0, "快速检查 5 分钟负载阈值")
	cmd.Flags().Float64("load15", 0, "快速检查 15 分钟负载阈值")

	return cmd
}

// runQuickCheck 执行快速检查模式。
func runQuickCheck(cmd *cobra.Command, checker alert.Checker, output string) error {
	type quickCheck struct {
		Metric    string  `json:"metric"`
		Value     float64 `json:"value"`
		Operator  string  `json:"operator"`
		Threshold float64 `json:"threshold"`
		Pass      bool    `json:"pass"`
		Message   string  `json:"message"`
	}

	var checks []quickCheck

	// CPU 检查
	if v, _ := cmd.Flags().GetFloat64("cpu"); v > 0 {
		val, _ := checker.Check("cpu_percent")
		msg := "正常"
		pass := val <= v
		if !pass {
			msg = "异常"
		}
		checks = append(checks, quickCheck{
			Metric: "cpu_percent", Value: val, Operator: "<=",
			Threshold: v, Pass: pass, Message: fmt.Sprintf("CPU: %.1f%% (阈值: %.1f%%) %s", val, v, msg),
		})
	}

	// 内存检查
	if v, _ := cmd.Flags().GetFloat64("memory"); v > 0 {
		val, _ := checker.Check("memory_percent")
		pass := val <= v
		msg := "正常"
		if !pass {
			msg = "异常"
		}
		checks = append(checks, quickCheck{
			Metric: "memory_percent", Value: val, Operator: "<=",
			Threshold: v, Pass: pass, Message: fmt.Sprintf("内存: %.1f%% (阈值: %.1f%%) %s", val, v, msg),
		})
	}

	// 磁盘检查
	if v, _ := cmd.Flags().GetFloat64("disk"); v > 0 {
		val, _ := checker.Check("disk_percent")
		pass := val <= v
		msg := "正常"
		if !pass {
			msg = "异常"
		}
		checks = append(checks, quickCheck{
			Metric: "disk_percent", Value: val, Operator: "<=",
			Threshold: v, Pass: pass, Message: fmt.Sprintf("磁盘: %.1f%% (阈值: %.1f%%) %s", val, v, msg),
		})
	}

	// 负载检查
	for _, name := range []string{"load1", "load5", "load15"} {
		if v, _ := cmd.Flags().GetFloat64(name); v > 0 {
			val, _ := checker.Check(name)
			pass := val <= v
			msg := "正常"
			if !pass {
				msg = "异常"
			}
			checks = append(checks, quickCheck{
				Metric: name, Value: val, Operator: "<=",
				Threshold: v, Pass: pass, Message: fmt.Sprintf("负载 %s: %.2f (阈值: %.2f) %s", name, val, v, msg),
			})
		}
	}

	if output == "json" {
		data, _ := json.MarshalIndent(checks, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
	} else {
		allPass := true
		for _, c := range checks {
			if c.Pass {
				fmt.Fprintf(cmd.OutOrStdout(), "✅ %s\n", c.Message)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "❌ %s\n", c.Message)
				allPass = false
			}
		}
		if !allPass {
			return fmt.Errorf("存在异常指标")
		}
	}

	return nil
}

// newAlertRulesCmd 创建 rules 子命令。
func newAlertRulesCmd() *cobra.Command {
	var config string

	cmd := &cobra.Command{
		Use:   "rules",
		Short: "管理告警规则",
		Long:  `列出、验证和管理告警规则配置。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if config == "" {
				homeDir, _ := os.UserHomeDir()
				config = filepath.Join(homeDir, ".opsxcli", "alert_rules.yaml")
			}

			ruleConfig, err := alert.LoadRules(config)
			if err != nil {
				return fmt.Errorf("加载规则配置失败: %w", err)
			}

			if len(ruleConfig.Rules) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "没有配置告警规则")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "=== 告警规则 (%d 条) ===\n\n", len(ruleConfig.Rules))
			for i, rule := range ruleConfig.Rules {
				status := "✅ 启用"
				if !rule.Enabled {
					status = "⏸️ 禁用"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d. [%s] %s\n", i+1, status, rule.Name)
				fmt.Fprintf(cmd.OutOrStdout(), "   描述: %s\n", rule.Description)
				fmt.Fprintf(cmd.OutOrStdout(), "   检查: %s\n", rule.Check)
				fmt.Fprintf(cmd.OutOrStdout(), "   间隔: %s\n", rule.Interval)
				fmt.Fprintf(cmd.OutOrStdout(), "   级别: %s\n", rule.Level)
				fmt.Fprintf(cmd.OutOrStdout(), "   渠道: %s\n\n", strings.Join(rule.Channels, ", "))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&config, "config", "c", "", "告警规则配置文件路径")
	return cmd
}

// newAlertHistoryCmd 创建 history 子命令。
func newAlertHistoryCmd() *cobra.Command {
	var (
		limit    int
		ruleName string
		level    string
		dataDir  string
		output   string
	)

	cmd := &cobra.Command{
		Use:   "history",
		Short: "查询告警历史",
		Long:  `查询和展示告警历史记录。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			hm, err := alert.NewHistoryManager(dataDir)
			if err != nil {
				return fmt.Errorf("创建历史管理器失败: %w", err)
			}

			filter := &alert.HistoryFilter{
				RuleName: ruleName,
				Limit:    limit,
			}
			if level != "" {
				filter.Level = alert.Level(level)
			}

			entries, err := hm.Query(filter)
			if err != nil {
				return fmt.Errorf("查询历史失败: %w", err)
			}

			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "没有告警历史记录")
				return nil
			}

			if output == "json" {
				data, _ := json.MarshalIndent(entries, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "=== 告警历史 (最近 %d 条) ===\n\n", len(entries))
				for _, e := range entries {
					stateIcon := "🔴"
					if e.State == alert.StateResolved {
						stateIcon = "🟢"
					} else if e.State == alert.StateSilenced {
						stateIcon = "🔇"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] %s - %s\n", stateIcon, e.Level, e.RuleName, e.Message)
					fmt.Fprintf(cmd.OutOrStdout(), "   值: %.1f (阈值: %.1f) @ %s\n\n", e.Value, e.Threshold,
						e.Timestamp.Format("2006-01-02 15:04:05"))
				}
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "显示条数")
	cmd.Flags().StringVarP(&ruleName, "rule", "r", "", "按规则名称过滤")
	cmd.Flags().StringVarP(&level, "level", "l", "", "按级别过滤 (info/warning/error/critical)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "数据目录 (默认 ~/.opsxcli/alert)")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "输出格式 (text/json)")

	return cmd
}

// newAlertSilenceCmd 创建 silence 子命令。
func newAlertSilenceCmd() *cobra.Command {
	var (
		duration string
		reason   string
		ruleName string
		level    string
		dataDir  string
	)

	cmd := &cobra.Command{
		Use:   "silence",
		Short: "管理告警静默规则",
		Long: `管理告警静默规则，可以按规则名称、级别或标签匹配告警并临时静默。

示例：
  opsxcli alert silence --rule high_cpu --duration 1h --reason "维护窗口"
  opsxcli alert silence --level critical --duration 30m --reason "已知问题"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			sm, err := alert.NewSilenceManager(dataDir)
			if err != nil {
				return fmt.Errorf("创建静默管理器失败: %w", err)
			}

			// 无参数时列出所有静默规则
			if !cmd.Flags().Changed("duration") {
				rules := sm.List()
				if len(rules) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "没有活跃的静默规则")
					return nil
				}

				fmt.Fprintf(cmd.OutOrStdout(), "=== 活跃静默规则 (%d 条) ===\n\n", len(rules))
				for _, r := range rules {
					fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\n", r.ID)
					if r.MatchName != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "  匹配规则: %s\n", r.MatchName)
					}
					if r.MatchLevel != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "  匹配级别: %s\n", r.MatchLevel)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  原因: %s\n", r.Reason)
					fmt.Fprintf(cmd.OutOrStdout(), "  截止: %s\n\n", r.Until.Format("2006-01-02 15:04:05"))
				}
				return nil
			}

			// 添加静默规则
			if reason == "" {
				return fmt.Errorf("静默原因不能为空，使用 --reason 指定")
			}

			dur, err := time.ParseDuration(duration)
			if err != nil {
				return fmt.Errorf("无法解析时长 %q: %w", duration, err)
			}

			silenceRule := &alert.SilenceRule{
				MatchName: ruleName,
				Reason:    reason,
				Until:     time.Now().Add(dur),
			}
			if level != "" {
				silenceRule.MatchLevel = alert.Level(level)
			}

			if err := sm.Add(silenceRule); err != nil {
				return fmt.Errorf("添加静默规则失败: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "静默规则已添加 (截止: %s)\n", silenceRule.Until.Format("2006-01-02 15:04:05"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&duration, "duration", "d", "", "静默时长 (如 1h, 30m)")
	cmd.Flags().StringVar(&reason, "reason", "", "静默原因")
	cmd.Flags().StringVarP(&ruleName, "rule", "r", "", "匹配的规则名称")
	cmd.Flags().StringVarP(&level, "level", "l", "", "匹配的告警级别")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "数据目录")

	return cmd
}

// newAlertInitCmd 创建 init 子命令。
func newAlertInitCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "生成默认告警配置文件",
		Long:  `生成默认的告警规则配置文件模板，可在此基础上自定义规则。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output == "" {
				homeDir, _ := os.UserHomeDir()
				output = filepath.Join(homeDir, ".opsxcli", "alert_rules.yaml")
			}

			config := alert.DefaultRuleConfig()
			if err := alert.SaveRules(output, config); err != nil {
				return fmt.Errorf("生成配置文件失败: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "告警配置文件已生成: %s\n", output)
			fmt.Fprintln(cmd.OutOrStdout(), "请编辑配置文件中的 webhook URL 和规则参数。")
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "输出文件路径 (默认 ~/.opsxcli/alert_rules.yaml)")
	return cmd
}
