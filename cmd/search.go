package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"opsxcli/internal/output"
)

func init() {
	RegisterCommand("search", "工具", "搜索已注册命令", NewSearchCmd)
}

// searchResult 搜索结果条目
type searchResult struct {
	Name        string
	Category    string
	Description string
	Aliases     []string
}

// NewSearchCmd 创建 search 命令
func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <keyword> [keyword...]",
		Short: "搜索已注册命令",
		Long: `模糊搜索所有已注册命令，支持按命令名、别名、描述、分类匹配。

支持多个关键词 AND 匹配，即所有关键词都匹配时才显示结果。

示例:
  # 搜索包含 "数据库" 的命令
  opsxcli search 数据库

  # 搜索同时包含 "网络" 和 "测试" 的命令
  opsxcli search 网络 测试

  # 以 JSON 格式输出搜索结果
  opsxcli search 网络监控 --output json`,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			// 搜索匹配结果
			results := searchCommands(args)

			if len(results) == 0 {
				fmt.Println(color.YellowString("未找到匹配的命令"))
				return nil
			}

			// 根据输出格式输出结果
			if output.IsJSON() {
				output.JSON(results)
			} else {
				printSearchResults(results, args)
			}
			return nil
		},
	}

	return cmd
}

// searchCommands 搜索命令，支持多关键词 AND 匹配
func searchCommands(keywords []string) []searchResult {
	var results []searchResult

	// 预处理关键词：全部转小写
	lowerKeywords := make([]string, len(keywords))
	for i, kw := range keywords {
		lowerKeywords[i] = strings.ToLower(kw)
	}

	// 遍历注册表中的命令
	for name, entry := range GetRegisteredCommands() {
		// 获取命令的别名（通过 Factory 创建临时命令来获取）
		aliases := getCommandAliases(name, entry)

		// 构建可搜索的文本
		searchText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
			name,
			strings.Join(aliases, " "),
			entry.Category,
			entry.Description,
		))

		// AND 匹配：所有关键词都必须匹配
		matched := true
		for _, kw := range lowerKeywords {
			if !strings.Contains(searchText, kw) {
				matched = false
				break
			}
		}

		if matched {
			results = append(results, searchResult{
				Name:        name,
				Category:    entry.Category,
				Description: entry.Description,
				Aliases:     aliases,
			})
		}
	}

	// 按分类排序，同类按名称排序
	sort.Slice(results, func(i, j int) bool {
		if results[i].Category != results[j].Category {
			return results[i].Category < results[j].Category
		}
		return results[i].Name < results[j].Name
	})

	return results
}

// getCommandAliases 获取命令的别名
func getCommandAliases(name string, entry CommandEntry) []string {
	// 创建临时命令获取别名信息
	c := entry.Factory()
	if c != nil {
		return c.Aliases
	}
	return nil
}

// printSearchResults 以表格形式输出搜索结果
func printSearchResults(results []searchResult, keywords []string) {
	// 表头
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Printf("\n%s 共找到 %d 个匹配命令 (关键词: %s)\n\n",
		green("✓"),
		len(results),
		yellow(strings.Join(keywords, " + ")),
	)

	// 按分类分组输出
	currentCategory := ""
	for _, r := range results {
		if r.Category != currentCategory {
			currentCategory = r.Category
			fmt.Printf("  %s\n", yellow("["+currentCategory+"]"))
		}

		aliasStr := ""
		if len(r.Aliases) > 0 {
			aliasStr = fmt.Sprintf(" (%s)", strings.Join(r.Aliases, ", "))
		}
		fmt.Printf("    %-16s %s%s\n", cyan(r.Name), r.Description, color.HiBlackString(aliasStr))
	}

	// 底部提示
	fmt.Printf("\n  %s %s %s\n",
		color.HiBlackString("使用"),
		cyan("opsxcli <command> --help"),
		color.HiBlackString("查看命令详情"),
	)

	// 确保输出刷新
	os.Stdout.Sync()
}
