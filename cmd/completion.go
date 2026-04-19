package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	RegisterCommand("completion", "工具", "生成Shell自动补全脚本", NewCompletionCmd)
}

// NewCompletionCmd 创建 completion 命令
// 使用 Cobra 原生 completion 机制，支持 bash、zsh、fish
func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "生成 Shell 自动补全脚本",
		Long: `为指定的 Shell 生成自动补全脚本，方便快速输入命令和参数。

支持的 Shell:
  bash - 生成 bash 补全脚本
  zsh  - 生成 zsh 补全脚本
  fish - 生成 fish 补全脚本

示例:
  # 为 bash 生成补全脚本并加载
  opsxcli completion bash > /etc/bash_completion.d/opsxcli
  source /etc/bash_completion.d/opsxcli

  # 为 zsh 生成补全脚本
  opsxcli completion zsh > "${fpath[1]}/_opsxcli"

  # 为 fish 生成补全脚本
  opsxcli completion fish > ~/.config/fish/completions/opsxcli.fish`,
		ValidArgs: []string{"bash", "zsh", "fish"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 获取根命令
			rootCmd := cmd.Root()
			shell := args[0]

			switch shell {
			case "bash":
				return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			default:
				cmd.Help()
			}
			return nil
		},
		DisableFlagsInUseLine: true,
	}

	return cmd
}
