package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// psql analyze 子命令参数解析测试
// ============================================================================

// newTestPsqlAnalyzeCmd 创建用于测试的 analyze 子命令（不执行 RunE）
func newTestPsqlAnalyzeCmd() *cobra.Command {
	return newPsqlAnalyzeCmd()
}

// TestPsqlAnalyzeCmd_FlagRegistration 验证所有 flag 已正确注册
func TestPsqlAnalyzeCmd_FlagRegistration(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()

	// 验证连接参数 flag
	testCases := []struct {
		name      string
		shorthand string
	}{
		{"host", "h"},
		{"port", "P"},
		{"user", "U"},
		{"password", "W"},
		{"database", "d"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tc.name)
			require.NotNil(t, f, "flag %s 应该被注册", tc.name)
			assert.Equal(t, tc.shorthand, f.Shorthand, "flag %s 的 shorthand 应该是 %s", tc.name, tc.shorthand)
		})
	}

	// 验证 analyze 特有 flag
	analyzeFlags := []string{"active", "locks", "all", "long-query-time"}
	for _, name := range analyzeFlags {
		t.Run(name, func(t *testing.T) {
			f := cmd.Flags().Lookup(name)
			require.NotNil(t, f, "flag %s 应该被注册", name)
		})
	}
}

// TestPsqlAnalyzeCmd_DefaultValues 验证 flag 默认值
func TestPsqlAnalyzeCmd_DefaultValues(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()

	host, _ := cmd.Flags().GetString("host")
	assert.Equal(t, "127.0.0.1", host, "默认主机应该是 127.0.0.1")

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 5432, port, "默认端口应该是 5432")

	user, _ := cmd.Flags().GetString("user")
	assert.Equal(t, "postgres", user, "默认用户应该是 postgres")

	active, _ := cmd.Flags().GetBool("active")
	assert.False(t, active, "--active 默认应该是 false")

	locks, _ := cmd.Flags().GetBool("locks")
	assert.False(t, locks, "--locks 默认应该是 false")

	all, _ := cmd.Flags().GetBool("all")
	assert.False(t, all, "--all 默认应该是 false")

	lqt, _ := cmd.Flags().GetInt("long-query-time")
	assert.Equal(t, 5, lqt, "--long-query-time 默认应该是 5")
}

// TestPsqlAnalyzeCmd_PasswordNoOptDefVal 验证密码 flag 支持可选参数
func TestPsqlAnalyzeCmd_PasswordNoOptDefVal(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()

	pwdFlag := cmd.Flags().Lookup("password")
	require.NotNil(t, pwdFlag)
	assert.Equal(t, "ASK", pwdFlag.NoOptDefVal, "密码 flag 的 NoOptDefVal 应该是 ASK")
}

// TestPsqlAnalyzeCmd_ParseActiveFlag 验证 --active 参数解析
func TestPsqlAnalyzeCmd_ParseActiveFlag(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()
	cmd.SetArgs([]string{"--active"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }

	err := cmd.Execute()
	require.NoError(t, err)

	active, _ := cmd.Flags().GetBool("active")
	assert.True(t, active)
}

// TestPsqlAnalyzeCmd_ParseLocksFlag 验证 --locks 参数解析
func TestPsqlAnalyzeCmd_ParseLocksFlag(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()
	cmd.SetArgs([]string{"--locks"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }

	err := cmd.Execute()
	require.NoError(t, err)

	locks, _ := cmd.Flags().GetBool("locks")
	assert.True(t, locks)
}

// TestPsqlAnalyzeCmd_ParseAllFlags 验证同时指定多个 flag
func TestPsqlAnalyzeCmd_ParseAllFlags(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()
	cmd.SetArgs([]string{"--active", "--locks", "--long-query-time", "30"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }

	err := cmd.Execute()
	require.NoError(t, err)

	active, _ := cmd.Flags().GetBool("active")
	locks, _ := cmd.Flags().GetBool("locks")
	lqt, _ := cmd.Flags().GetInt("long-query-time")

	assert.True(t, active)
	assert.True(t, locks)
	assert.Equal(t, 30, lqt)
}

// TestPsqlAnalyzeCmd_ParseConnectionParams 验证连接参数解析
func TestPsqlAnalyzeCmd_ParseConnectionParams(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()
	cmd.SetArgs([]string{
		"--host", "192.168.1.100",
		"--port", "5433",
		"--user", "admin",
		"-W=secret",
		"--database", "testdb",
	})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }

	err := cmd.Execute()
	require.NoError(t, err)

	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	password, _ := cmd.Flags().GetString("password")
	database, _ := cmd.Flags().GetString("database")

	assert.Equal(t, "192.168.1.100", host)
	assert.Equal(t, 5433, port)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "secret", password)
	assert.Equal(t, "testdb", database)
}

// TestPsqlAnalyzeCmd_ParseShorthandFlags 验证 shorthand flag 解析
func TestPsqlAnalyzeCmd_ParseShorthandFlags(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()
	cmd.SetArgs([]string{"-h", "10.0.0.1", "-P", "5432", "-U", "pguser", "-d", "mydb"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }

	err := cmd.Execute()
	require.NoError(t, err)

	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	user, _ := cmd.Flags().GetString("user")
	database, _ := cmd.Flags().GetString("database")

	assert.Equal(t, "10.0.0.1", host)
	assert.Equal(t, 5432, port)
	assert.Equal(t, "pguser", user)
	assert.Equal(t, "mydb", database)
}

// TestPsqlAnalyzeCmd_ParsePasswordShorthand 验证 -W shorthand 解析
func TestPsqlAnalyzeCmd_ParsePasswordShorthand(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()
	cmd.SetArgs([]string{"-W=mypassword"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }

	err := cmd.Execute()
	require.NoError(t, err)

	password, _ := cmd.Flags().GetString("password")
	assert.Equal(t, "mypassword", password)
}

// TestPsqlAnalyzeCmd_UseAndShort 验证命令基本信息
func TestPsqlAnalyzeCmd_UseAndShort(t *testing.T) {
	cmd := newTestPsqlAnalyzeCmd()

	assert.Equal(t, "analyze", cmd.Use)
	assert.Contains(t, cmd.Short, "PostgreSQL")
	assert.Contains(t, cmd.Short, "性能分析")
	assert.True(t, cmd.SilenceUsage)
}
