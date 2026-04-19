package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"opsxcli/internal/sshconfig"
)

func init() {
	RegisterCommand("ssh-config", "网络", "SSH配置管理", NewSSHConfigCmd)
}

// NewSSHConfigCmd 创建 ssh-config 命令
func NewSSHConfigCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "ssh-config",
		Short: "SSH配置管理 - 查看 ~/.ssh/config 中的主机配置",
		Long: `ssh-config - 管理 SSH 配置文件中的主机别名

支持列出所有配置的主机，以及查看单个主机的详细配置信息。`,
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", sshconfig.DefaultPath(), "指定 SSH 配置文件路径")

	cmd.AddCommand(newSSHConfigListCmd(&configPath))
	cmd.AddCommand(newSSHConfigShowCmd(&configPath))

	return cmd
}

func newSSHConfigListCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出 ~/.ssh/config 中配置的所有主机",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := sshconfig.Parse(*configPath)
			if err != nil {
				return fmt.Errorf("读取 SSH 配置失败: %w", err)
			}

			if len(cfg.Hosts) == 0 {
				fmt.Println(color.YellowString("未在 ~/.ssh/config 中发现主机配置"))
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, color.CyanString("别名\t主机地址\t用户\t端口\t私钥路径"))
			fmt.Fprintln(w, color.HiBlackString("────\t────────\t────\t────\t────────"))

			for _, h := range cfg.Hosts {
				hostName := h.HostName
				if hostName == "" {
					hostName = h.Alias
				}
				identity := h.IdentityFile
				if identity == "" {
					identity = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					color.GreenString(h.Alias),
					hostName,
					h.User,
					h.Port,
					identity,
				)
			}
			w.Flush()
			return nil
		},
	}
}

func newSSHConfigShowCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "show <alias>",
		Short: "显示指定主机的详细配置",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := sshconfig.Parse(*configPath)
			if err != nil {
				return fmt.Errorf("读取 SSH 配置失败: %w", err)
			}

			h := cfg.GetHost(args[0])
			if h == nil {
				return fmt.Errorf("未找到主机别名: %s", args[0])
			}

			hostName := h.HostName
			if hostName == "" {
				hostName = h.Alias
			}

			fmt.Printf("%s 主机配置详情\n\n", color.CyanString("📋"))
			fmt.Printf("  %-12s %s\n", "Alias:", color.GreenString(h.Alias))
			fmt.Printf("  %-12s %s\n", "HostName:", hostName)
			fmt.Printf("  %-12s %s\n", "User:", h.User)
			fmt.Printf("  %-12s %d\n", "Port:", h.Port)
			fmt.Printf("  %-12s %s\n", "IdentityFile:", h.IdentityFile)
			return nil
		},
	}
}
