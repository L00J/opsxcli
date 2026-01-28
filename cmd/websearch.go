package cmd

import (
	"strings"

	"opsxcli/plugins/websearch"

	"github.com/spf13/cobra"
)

// NewWebSearchCmd 创建 websearch 命令
func NewWebSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "websearch <query>",
		Short: "Web搜索工具(支持国内外多个搜索引擎)",
		Long: `在多个搜索引擎中搜索关键词

支持的搜索引擎:
  国内: baidu, sogou, so360, bing_cn
  海外: google, bing, duckduckgo, yahoo

所有搜索引擎都无需 API Token,直接模拟浏览器访问。

示例:
  # 使用默认搜索引擎(百度+Google)
  opsxcli websearch "语雀 API 文档"

  # 指定多个搜索引擎
  opsxcli websearch "Claude Code" --engines google,bing,baidu

  # 保存搜索结果 HTML
  opsxcli websearch "Kubernetes" --engines baidu --save-html

  # 列出所有可用搜索引擎
  opsxcli websearch --list`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 列出搜索引擎
			list, _ := cmd.Flags().GetBool("list")
			if list {
				websearch.ListEngines()
				return nil
			}

			// 检查是否提供了搜索关键词
			if len(args) == 0 {
				return cmd.Help()
			}

			query := strings.Join(args, " ")
			enginesStr, _ := cmd.Flags().GetString("engines")
			saveHTML, _ := cmd.Flags().GetBool("save-html")

			// 解析搜索引擎列表
			var engines []string
			if enginesStr != "" {
				engines = strings.Split(enginesStr, ",")
				for i := range engines {
					engines[i] = strings.TrimSpace(engines[i])
				}
			}

			return websearch.Search(query, engines, saveHTML)
		},
	}

	cmd.Flags().StringP("engines", "e", "", "搜索引擎列表(逗号分隔,如: baidu,google)")
	cmd.Flags().BoolP("save-html", "s", false, "保存搜索结果 HTML")
	cmd.Flags().BoolP("list", "l", false, "列出所有可用的搜索引擎")

	return cmd
}
