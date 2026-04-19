package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"opsxcli/internal/sshconfig"
	"opsxcli/plugins/ssh"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("ssh", "网络", "SSH远程连接与执行", NewSSHCmd)
}

// NewSSHCmd 创建SSH命令
func NewSSHCmd() *cobra.Command {
	var configPath string
	var listConfig bool

	rootCmd := &cobra.Command{
		Use:   "ssh [user@]host [command]",
		Short: "SSH连接、命令执行、文件传输、端口转发",
		Long: `SSH工具支持以下功能：
  - 交互式登录
  - 远程命令执行
  - SFTP文件传输（上传/下载）
  - 端口转发（本地/远程/动态）

使用示例:
  opsxcli ssh root@10.10.10.156              # 交互式登录
  opsxcli ssh root@10.10.10.156 "ls -la"     # 执行命令
  opsxcli ssh put local.txt root@host:/tmp/  # 上传文件
  opsxcli ssh get root@host:/tmp/file .      # 下载文件
  opsxcli ssh forward local 8080:80 root@host # 端口转发
  opsxcli ssh prod                           # 从 ~/.ssh/config 解析 prod 配置
  opsxcli ssh --list-config                  # 列出 ~/.ssh/config 中的所有主机`,
		Args: func(cmd *cobra.Command, args []string) error {
			if listConfig {
				return nil
			}
			if len(args) < 1 {
				return fmt.Errorf("至少需要指定目标主机")
			}
			return nil
		},
		SilenceUsage:  true,  // 错误时不显示 Usage
		SilenceErrors: false, // 显示错误信息
		RunE: func(cmd *cobra.Command, args []string) error {
			if listConfig {
				return listSSHConfig(configPath)
			}

			target := args[0]
			keyPath, _ := cmd.Flags().GetString("key")
			port, _ := cmd.Flags().GetInt("port")
			password, _ := cmd.Flags().GetString("password")

			// 解析显式指定的 user@host
			explicitUser := ""
			hostPart := target
			if strings.Contains(target, "@") {
				parts := strings.SplitN(target, "@", 2)
				explicitUser = parts[0]
				hostPart = parts[1]
			}

			// 尝试从 SSH 配置文件解析主机配置
			cfg, err := sshconfig.Parse(configPath)
			if err != nil {
				return fmt.Errorf("读取 SSH 配置失败: %w", err)
			}

			if h := cfg.GetHost(hostPart); h != nil {
				// 使用配置中的 HostName，若未设置则回退到别名本身
				resolvedHost := h.HostName
				if resolvedHost == "" {
					resolvedHost = hostPart
				}

				// 用户优先级：命令行显式指定 > 配置文件 > 默认 root
				resolvedUser := h.User
				if explicitUser != "" {
					resolvedUser = explicitUser
				}
				if resolvedUser == "" {
					resolvedUser = "root"
				}

				target = resolvedUser + "@" + resolvedHost

				// 端口优先级：命令行显式指定 > 配置文件 > 默认 22
				if !cmd.Flags().Changed("port") && h.Port != 0 {
					port = h.Port
				}

				// 私钥优先级：命令行显式指定 > 配置文件
				if !cmd.Flags().Changed("key") && h.IdentityFile != "" {
					keyPath = h.IdentityFile
				}
			}

			// 如果只有一个参数，执行交互式连接
			if len(args) == 1 {
				return ssh.Connect(target, keyPath, port, password)
			}

			// 如果有多个参数，执行命令
			command := strings.Join(args[1:], " ")
			return ssh.ExecCommand(target, keyPath, port, password, command)
		},
	}

	// 添加全局参数
	rootCmd.Flags().StringP("key", "i", "", "SSH私钥路径")
	rootCmd.Flags().IntP("port", "p", 22, "SSH端口")
	rootCmd.Flags().StringP("password", "P", "", "密码（不指定则提示输入）")
	rootCmd.Flags().StringVar(&configPath, "config", sshconfig.DefaultPath(), "指定 SSH 配置文件路径")
	rootCmd.Flags().BoolVar(&listConfig, "list-config", false, "列出 SSH 配置文件中的所有主机")

	// 添加子命令（connect 和 exec 已通过主命令支持，保留用于向后兼容）
	rootCmd.AddCommand(
		ssh.NewPutCmd(),
		ssh.NewGetCmd(),
		ssh.NewForwardCmd(),
	)

	return rootCmd
}

func listSSHConfig(configPath string) error {
	cfg, err := sshconfig.Parse(configPath)
	if err != nil {
		return fmt.Errorf("读取 SSH 配置失败: %w", err)
	}

	if len(cfg.Hosts) == 0 {
		fmt.Println(color.YellowString("未在 SSH 配置文件中发现主机配置"))
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
}
