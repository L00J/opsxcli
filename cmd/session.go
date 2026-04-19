package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"opsxcli/internal/agentv2/session"
)

func init() {
	RegisterCommand("session", "AI", "会话管理", NewSessionCmd)
}

// NewSessionCmd 创建 session 命令
func NewSessionCmd() *cobra.Command {
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "会话管理 - 管理 AI Agent 对话会话",
		Long: `session - 会话管理

管理 AI Agent 的对话会话，支持列表、恢复、导出等操作。

示例：
  opsxcli session list                # 列出所有会话
  opsxcli session resume <id>         # 恢复会话
  opsxcli session export <id>         # 导出为 Markdown
  opsxcli session delete <id>         # 删除会话
  opsxcli session rename <id> <title> # 重命名会话`,
	}

	sessionCmd.AddCommand(newSessionListCmd())
	sessionCmd.AddCommand(newSessionResumeCmd())
	sessionCmd.AddCommand(newSessionExportCmd())
	sessionCmd.AddCommand(newSessionDeleteCmd())
	sessionCmd.AddCommand(newSessionRenameCmd())

	return sessionCmd
}

// initSessionManager 初始化 V2 会话管理器
func initSessionManager() (session.Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("获取用户主目录失败: %w", err)
	}

	sessionDir := filepath.Join(homeDir, ".opsxcli", "agent", "sessions")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("创建会话目录失败: %w", err)
	}

	store, err := session.NewJSONLStore(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("创建会话存储失败: %w", err)
	}

	return session.NewManager(store), nil
}

// newSessionListCmd 列出会话
func newSessionListCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有会话",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := initSessionManager()
			if err != nil {
				return err
			}

			sessions, err := mgr.List(limit)
			if err != nil {
				return fmt.Errorf("获取会话列表失败: %w", err)
			}

			if len(sessions) == 0 {
				fmt.Println(color.YellowString("没有找到会话"))
				return nil
			}

			// 使用 tabwriter 格式化输出
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, color.CyanString("ID\tTITLE\tPROVIDER\tMODEL\tUPDATED\tMESSAGES"))
			fmt.Fprintln(w, color.HiBlackString("──\t─────\t────────\t─────\t───────\t────────"))

			for _, sess := range sessions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\n",
					color.GreenString(sess.ID),
					sess.Title,
					sess.Provider,
					sess.Model,
					sess.UpdatedAt.Format("2006-01-02 15:04"),
					sess.MessageCount,
				)
			}

			w.Flush()
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "显示的会话数量")

	return cmd
}

// newSessionResumeCmd 恢复会话
func newSessionResumeCmd() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "resume <session-id>",
		Short: "恢复指定会话并继续对话",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			mgr, err := initSessionManager()
			if err != nil {
				return err
			}

			sess, messages, err := mgr.Load(sessionID)
			if err != nil {
				return fmt.Errorf("加载会话失败: %w", err)
			}

			// 显示会话信息
			fmt.Printf("📝 %s\n", color.CyanString(sess.Title))
			fmt.Printf("   Provider: %s | Model: %s\n", sess.Provider, sess.Model)
			fmt.Printf("   Messages: %d | Updated: %s\n\n",
				len(messages),
				sess.UpdatedAt.Format("2006-01-02 15:04:05"),
			)

			// 显示历史消息
			for _, msg := range messages {
				var roleColor *color.Color
				var roleIcon string

				switch msg.Role {
				case "user":
					roleColor = color.New(color.FgGreen)
					roleIcon = "👤"
				case "assistant":
					roleColor = color.New(color.FgCyan)
					roleIcon = "🤖"
				case "system":
					roleColor = color.New(color.FgYellow)
					roleIcon = "⚙️"
				case "tool":
					roleColor = color.New(color.FgWhite)
					roleIcon = "🔧"
				default:
					roleColor = color.New(color.FgWhite)
					roleIcon = "·"
				}

				fmt.Printf("%s %s:\n%s\n\n",
					roleIcon,
					roleColor.Sprint(msg.Role),
					msg.Content,
				)
			}

			_ = provider // 保留 provider flag 供后续使用
			fmt.Println(color.YellowString("提示: 使用 'opsxcli agent --resume " + sessionID + "' 继续对话"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "使用的 LLM 提供商 (覆盖会话配置)")

	return cmd
}

// newSessionExportCmd 导出会话
func newSessionExportCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "export <session-id>",
		Short: "导出会话为 Markdown 文件",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			mgr, err := initSessionManager()
			if err != nil {
				return err
			}

			markdown, err := mgr.ExportMarkdown(sessionID)
			if err != nil {
				return fmt.Errorf("导出会话失败: %w", err)
			}

			// 保存到文件
			if output == "" {
				output = fmt.Sprintf("session_%s.md", sessionID)
			}

			err = os.WriteFile(output, []byte(markdown), 0644)
			if err != nil {
				return fmt.Errorf("保存文件失败: %w", err)
			}

			fmt.Printf("%s 会话已导出到: %s\n",
				color.GreenString("✓"),
				color.CyanString(output),
			)

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "输出文件路径 (默认: session_<id>.md)")

	return cmd
}

// newSessionDeleteCmd 删除会话
func newSessionDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <session-id>",
		Short: "删除指定会话",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

			mgr, err := initSessionManager()
			if err != nil {
				return err
			}

			err = mgr.Delete(sessionID)
			if err != nil {
				return fmt.Errorf("删除会话失败: %w", err)
			}

			fmt.Printf("%s 会话已删除: %s\n",
				color.GreenString("✓"),
				sessionID,
			)

			return nil
		},
	}
}

// newSessionRenameCmd 重命名会话
func newSessionRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <session-id> <new-title>",
		Short: "重命名会话",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			newTitle := args[1]

			mgr, err := initSessionManager()
			if err != nil {
				return err
			}

			err = mgr.UpdateTitle(sessionID, newTitle)
			if err != nil {
				return fmt.Errorf("重命名会话失败: %w", err)
			}

			fmt.Printf("%s 会话已重命名: %s → %s\n",
				color.GreenString("✓"),
				sessionID,
				color.CyanString(newTitle),
			)

			return nil
		},
	}
}
