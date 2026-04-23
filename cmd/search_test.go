package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// ===== searchCommands 测试 =====

func TestSearchCommands_SingleKeyword(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("mysql", "数据库", "MySQL 客户端工具", func() *cobra.Command {
		return &cobra.Command{Use: "mysql", Short: "MySQL 客户端工具"}
	})
	RegisterCommand("redis", "数据库", "Redis 客户端工具", func() *cobra.Command {
		return &cobra.Command{Use: "redis", Short: "Redis 客户端工具"}
	})
	RegisterCommand("ssh", "网络", "SSH 连接工具", func() *cobra.Command {
		return &cobra.Command{Use: "ssh", Short: "SSH 连接工具"}
	})

	results := searchCommands([]string{"数据库"})
	assert.Len(t, results, 2)
	names := []string{results[0].Name, results[1].Name}
	assert.Contains(t, names, "mysql")
	assert.Contains(t, names, "redis")
}

func TestSearchCommands_MultipleKeywords_AND(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("mysql", "数据库", "MySQL 客户端工具", func() *cobra.Command {
		return &cobra.Command{Use: "mysql"}
	})
	RegisterCommand("redis", "数据库", "Redis 客户端工具", func() *cobra.Command {
		return &cobra.Command{Use: "redis"}
	})
	RegisterCommand("ping", "网络", "网络连通性测试", func() *cobra.Command {
		return &cobra.Command{Use: "ping"}
	})

	// AND 匹配：必须同时包含 "数据库" 和 "客户端"
	results := searchCommands([]string{"数据库", "客户端"})
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Contains(t, r.Category, "数据库")
	}
}

func TestSearchCommands_NoMatch(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("mysql", "数据库", "MySQL 工具", func() *cobra.Command {
		return &cobra.Command{Use: "mysql"}
	})

	results := searchCommands([]string{"不存在的关键词"})
	assert.Empty(t, results)
}

func TestSearchCommands_CaseInsensitive(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("mysql", "DATABASE", "MySQL Tool", func() *cobra.Command {
		return &cobra.Command{Use: "mysql"}
	})

	results := searchCommands([]string{"database"})
	assert.Len(t, results, 1)
	assert.Equal(t, "mysql", results[0].Name)
}

func TestSearchCommands_SortedByCategoryAndName(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("zebra", "AAA", "Zebra tool", func() *cobra.Command {
		return &cobra.Command{Use: "zebra"}
	})
	RegisterCommand("alpha", "BBB", "Alpha tool", func() *cobra.Command {
		return &cobra.Command{Use: "alpha"}
	})
	RegisterCommand("beta", "AAA", "Beta tool", func() *cobra.Command {
		return &cobra.Command{Use: "beta"}
	})

	results := searchCommands([]string{"tool"})
	assert.Len(t, results, 3)
	// 先按分类排序，再按名称排序
	assert.Equal(t, "beta", results[0].Name)  // AAA 分类, beta
	assert.Equal(t, "zebra", results[1].Name) // AAA 分类, zebra
	assert.Equal(t, "alpha", results[2].Name) // BBB 分类, alpha
}

func TestSearchCommands_MatchByName(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("kubectl", "Kubernetes", "K8s 命令行工具", func() *cobra.Command {
		return &cobra.Command{Use: "kubectl"}
	})

	results := searchCommands([]string{"kubectl"})
	assert.Len(t, results, 1)
	assert.Equal(t, "kubectl", results[0].Name)
	assert.Equal(t, "Kubernetes", results[0].Category)
}

func TestSearchCommands_MatchByDescription(t *testing.T) {
	origRegistry := commandRegistry
	defer func() { commandRegistry = origRegistry }()

	commandRegistry = make(map[string]CommandEntry)
	RegisterCommand("sys", "监控", "系统监控 TUI", func() *cobra.Command {
		return &cobra.Command{Use: "sys"}
	})

	results := searchCommands([]string{"监控"})
	assert.Len(t, results, 1)
	assert.Equal(t, "sys", results[0].Name)
}

// ===== searchResult 测试 =====

func TestSearchResult_Struct(t *testing.T) {
	r := searchResult{
		Name:        "redis",
		Category:    "数据库",
		Description: "Redis 客户端",
		Aliases:     []string{"r"},
	}
	assert.Equal(t, "redis", r.Name)
	assert.Equal(t, "数据库", r.Category)
	assert.Equal(t, "Redis 客户端", r.Description)
	assert.Equal(t, []string{"r"}, r.Aliases)
}

// ===== getCommandAliases 测试 =====

func TestGetCommandAliases_WithAliases(t *testing.T) {
	entry := CommandEntry{
		Category: "测试",
		Factory: func() *cobra.Command {
			return &cobra.Command{
				Use:     "test",
				Aliases: []string{"t", "te"},
			}
		},
	}
	aliases := getCommandAliases("test", entry)
	assert.Equal(t, []string{"t", "te"}, aliases)
}

func TestGetCommandAliases_NoAliases(t *testing.T) {
	entry := CommandEntry{
		Category: "测试",
		Factory: func() *cobra.Command {
			return &cobra.Command{Use: "test"}
		},
	}
	aliases := getCommandAliases("test", entry)
	assert.Nil(t, aliases)
}
