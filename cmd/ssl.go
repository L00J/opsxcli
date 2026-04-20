package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"opsxcli/plugins/ssl"

	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("ssl", "网络", "SSL证书检查工具", NewSSLCmd)
}

// NewSSLCmd 创建 SSL 证书检查命令
func NewSSLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl <domain>",
		Short: "SSL证书检查工具",
		Long: `检查指定域名的 SSL 证书信息，包括有效期、签发者、签名算法等

功能特性：
  ✓ 证书有效期检查
  ✓ 到期预警提醒
  ✓ 完整证书链查看
  ✓ SAN 域名列表
  ✓ JSON 格式输出

示例：
  # 快速检查证书
  opsxcli ssl example.com

  # 指定端口
  opsxcli ssl example.com -p 8443

  # 显示完整证书链
  opsxcli ssl example.com --chain

  # 设置到期警告天数
  opsxcli ssl example.com --warn 60

  # JSON 格式输出
  opsxcli ssl example.com --json

  # 设置连接超时
  opsxcli ssl example.com --timeout 5s`,
		Args:          cobra.RangeArgs(0, 1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 无参数时显示帮助
			if len(args) == 0 {
				return cmd.Help()
			}

			domain := args[0]
			port, _ := cmd.Flags().GetInt("port")
			showChain, _ := cmd.Flags().GetBool("chain")
			warnDays, _ := cmd.Flags().GetInt("warn")
			outputJSON, _ := cmd.Flags().GetBool("json")
			timeout, _ := cmd.Flags().GetDuration("timeout")

			if timeout == 0 {
				timeout = 10 * time.Second
			}

			opts := &ssl.CheckOptions{
				Domain:    domain,
				Port:      port,
				Timeout:   timeout,
				ShowChain: showChain,
				WarnDays:  warnDays,
			}

			if showChain {
				chain, err := ssl.CheckChain(opts)
				if err != nil {
					return fmt.Errorf("证书链检查失败: %w", err)
				}

				if outputJSON {
					return outputJSONChain(chain)
				}

				fmt.Print(ssl.FormatChainOutput(chain))
				return nil
			}

			info, err := ssl.Check(opts)
			if err != nil {
				return fmt.Errorf("证书检查失败: %w", err)
			}

			if outputJSON {
				return outputJSONInfo(info)
			}

			fmt.Print(ssl.FormatOutput(info, warnDays))
			return nil
		},
	}

	cmd.Flags().IntP("port", "p", 443, "端口号")
	cmd.Flags().Bool("chain", false, "显示完整证书链")
	cmd.Flags().Int("warn", 30, "到期警告天数")
	cmd.Flags().Bool("json", false, "JSON 格式输出")
	cmd.Flags().Duration("timeout", 10*time.Second, "连接超时时间")

	return cmd
}

// outputJSONInfo 以 JSON 格式输出单个证书信息
func outputJSONInfo(info *ssl.CertificateInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// outputJSONChain 以 JSON 格式输出证书链
func outputJSONChain(chain []*ssl.CertificateInfo) error {
	data, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 序列化失败: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
