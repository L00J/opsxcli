package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== mysql analyze 子命令测试 ====================

func TestMySQLAnalyzeCmd_Registration(t *testing.T) {
	// 验证 analyze 子命令已注册到 mysql 父命令
	mysqlCmd := NewMySQLCmd()
	require.NotNil(t, mysqlCmd)

	// 查找 analyze 子命令
	analyzeCmd, _, err := mysqlCmd.Find([]string{"analyze"})
	require.NoError(t, err)
	assert.Equal(t, "analyze", analyzeCmd.Name())
	assert.NotNil(t, analyzeCmd)
}

func TestMySQLAnalyzeCmd_Flags(t *testing.T) {
	mysqlCmd := NewMySQLCmd()
	analyzeCmd, _, _ := mysqlCmd.Find([]string{"analyze"})

	// 验证特有 flag 已注册
	flags := analyzeCmd.Flags()

	// --sql flag
	sqlFlag := flags.Lookup("sql")
	assert.NotNil(t, sqlFlag)
	assert.Equal(t, "", sqlFlag.DefValue)

	// --normalize flag
	normalizeFlag := flags.Lookup("normalize")
	assert.NotNil(t, normalizeFlag)
	assert.Equal(t, "", normalizeFlag.DefValue)

	// --long-query-time flag
	lqtFlag := flags.Lookup("long-query-time")
	assert.NotNil(t, lqtFlag)
	assert.Equal(t, "10", lqtFlag.DefValue)

	// --indexes flag
	indexesFlag := flags.Lookup("indexes")
	assert.NotNil(t, indexesFlag)
	assert.Equal(t, "false", indexesFlag.DefValue)

	// --process flag
	processFlag := flags.Lookup("process")
	assert.NotNil(t, processFlag)
	assert.Equal(t, "false", processFlag.DefValue)

	// --format flag
	formatFlag := flags.Lookup("format")
	assert.NotNil(t, formatFlag)
	assert.Equal(t, "text", formatFlag.DefValue)
}

func TestMySQLAnalyzeCmd_SQLAnalysis(t *testing.T) {
	mysqlCmd := NewMySQLCmd()
	analyzeCmd, _, _ := mysqlCmd.Find([]string{"analyze"})

	// 设置 --sql 参数
	err := analyzeCmd.Flags().Set("sql", "SELECT * FROM users WHERE id = 1")
	require.NoError(t, err)

	// 验证参数能正确获取
	sqlInput, _ := analyzeCmd.Flags().GetString("sql")
	assert.Equal(t, "SELECT * FROM users WHERE id = 1", sqlInput)
}

func TestMySQLAnalyzeCmd_NormalizeFlag(t *testing.T) {
	mysqlCmd := NewMySQLCmd()
	analyzeCmd, _, _ := mysqlCmd.Find([]string{"analyze"})

	err := analyzeCmd.Flags().Set("normalize", "SELECT  *   FROM   users")
	require.NoError(t, err)

	normalizeInput, _ := analyzeCmd.Flags().GetString("normalize")
	assert.Equal(t, "SELECT  *   FROM   users", normalizeInput)
}

func TestMySQLAnalyzeCmd_DBFlags(t *testing.T) {
	mysqlCmd := NewMySQLCmd()
	analyzeCmd, _, _ := mysqlCmd.Find([]string{"analyze"})

	// 验证继承了数据库连接参数
	flags := analyzeCmd.Flags()

	host, _ := flags.GetString("host")
	assert.Equal(t, "127.0.0.1", host)

	port, _ := flags.GetInt("port")
	assert.Equal(t, 3306, port)

	user, _ := flags.GetString("user")
	assert.Equal(t, "root", user)
}

func TestMySQLAnalyzeCmd_LongQueryTimeFlag(t *testing.T) {
	mysqlCmd := NewMySQLCmd()
	analyzeCmd, _, _ := mysqlCmd.Find([]string{"analyze"})

	// 修改阈值
	err := analyzeCmd.Flags().Set("long-query-time", "30")
	require.NoError(t, err)

	lqt, _ := analyzeCmd.Flags().GetInt("long-query-time")
	assert.Equal(t, 30, lqt)
}
