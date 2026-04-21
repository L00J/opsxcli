package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// ===== NewPsqlCmd 测试 =====

func TestNewPsqlCmd_Basic(t *testing.T) {
	cmd := NewPsqlCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "psql", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "PostgreSQL")
}

func TestNewPsqlCmd_Aliases(t *testing.T) {
	cmd := NewPsqlCmd()
	assert.Contains(t, cmd.Aliases, "postgres", "psql 命令应有 postgres 别名")
}

func TestNewPsqlCmd_HasFlags(t *testing.T) {
	cmd := NewPsqlCmd()

	expectedFlags := []string{"host", "port", "user", "password", "database", "execute", "help"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "缺少 flag: %s", f)
	}
}

func TestNewPsqlCmd_FlagDefaults(t *testing.T) {
	cmd := NewPsqlCmd()

	host, _ := cmd.Flags().GetString("host")
	assert.Equal(t, "127.0.0.1", host, "host 默认值应为 127.0.0.1")

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 5432, port, "port 默认值应为 5432")

	user, _ := cmd.Flags().GetString("user")
	assert.Equal(t, "postgres", user, "user 默认值应为 postgres")

	password, _ := cmd.Flags().GetString("password")
	assert.Equal(t, "", password, "password 默认值应为空")

	database, _ := cmd.Flags().GetString("database")
	assert.Equal(t, "", database, "database 默认值应为空")

	execute, _ := cmd.Flags().GetString("execute")
	assert.Equal(t, "", execute, "execute 默认值应为空")
}

func TestNewPsqlCmd_FlagShorthands(t *testing.T) {
	cmd := NewPsqlCmd()

	assert.Equal(t, "h", cmd.Flags().Lookup("host").Shorthand)
	assert.Equal(t, "P", cmd.Flags().Lookup("port").Shorthand, "PostgreSQL 使用大写 -P")
	assert.Equal(t, "U", cmd.Flags().Lookup("user").Shorthand, "PostgreSQL 使用 -U")
	assert.Equal(t, "W", cmd.Flags().Lookup("password").Shorthand, "PostgreSQL 使用 -W")
	assert.Equal(t, "d", cmd.Flags().Lookup("database").Shorthand)
	assert.Equal(t, "c", cmd.Flags().Lookup("execute").Shorthand)
}

func TestNewPsqlCmd_PasswordNoOptDefVal(t *testing.T) {
	cmd := NewPsqlCmd()

	pwdFlag := cmd.Flags().Lookup("password")
	assert.NotNil(t, pwdFlag)
	assert.Equal(t, "ASK", pwdFlag.NoOptDefVal, "password flag 的 NoOptDefVal 应为 ASK")
}

func TestNewPsqlCmd_SilenceUsage(t *testing.T) {
	cmd := NewPsqlCmd()
	assert.True(t, cmd.SilenceUsage, "psql 命令应设置 SilenceUsage")
}

func TestNewPsqlCmd_HasRunE(t *testing.T) {
	cmd := NewPsqlCmd()
	assert.NotNil(t, cmd.RunE, "psql 命令应有 RunE")
}

// ===== psql dump 子命令测试 =====

func TestNewPsqlDumpCmd_Basic(t *testing.T) {
	cmd := NewPsqlCmd()
	var dumpCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "dump" {
			dumpCmd = sc
			break
		}
	}
	assert.NotNil(t, dumpCmd, "应存在 dump 子命令")
	assert.Equal(t, "dump", dumpCmd.Use)
	assert.NotEmpty(t, dumpCmd.Short)
}

func TestNewPsqlDumpCmd_HasFlags(t *testing.T) {
	cmd := NewPsqlCmd()
	var dumpCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "dump" {
			dumpCmd = sc
			break
		}
	}
	assert.NotNil(t, dumpCmd)

	// dump 特有参数
	dumpFlags := []string{"output", "format", "tables", "ignore-tables", "no-data", "no-schema"}
	for _, f := range dumpFlags {
		assert.NotNil(t, dumpCmd.Flags().Lookup(f), "dump 子命令缺少 flag: %s", f)
	}

	// 数据库连接参数
	dbFlags := []string{"host", "port", "user", "password", "database"}
	for _, f := range dbFlags {
		assert.NotNil(t, dumpCmd.Flags().Lookup(f), "dump 子命令缺少数据库连接 flag: %s", f)
	}
}

func TestNewPsqlDumpCmd_FlagDefaults(t *testing.T) {
	cmd := NewPsqlCmd()
	var dumpCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "dump" {
			dumpCmd = sc
			break
		}
	}
	assert.NotNil(t, dumpCmd)

	format, _ := dumpCmd.Flags().GetString("format")
	assert.Equal(t, "sql", format, "format 默认值应为 sql")

	output, _ := dumpCmd.Flags().GetString("output")
	assert.Equal(t, "", output, "output 默认值应为空")

	noData, _ := dumpCmd.Flags().GetBool("no-data")
	assert.False(t, noData, "no-data 默认值应为 false")

	noSchema, _ := dumpCmd.Flags().GetBool("no-schema")
	assert.False(t, noSchema, "no-schema 默认值应为 false")
}

// ===== psql restore 子命令测试 =====

func TestNewPsqlRestoreCmd_Basic(t *testing.T) {
	cmd := NewPsqlCmd()
	var restoreCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "restore" {
			restoreCmd = sc
			break
		}
	}
	assert.NotNil(t, restoreCmd, "应存在 restore 子命令")
	assert.Equal(t, "restore", restoreCmd.Use)
	assert.NotEmpty(t, restoreCmd.Short)
}

func TestNewPsqlRestoreCmd_HasFlags(t *testing.T) {
	cmd := NewPsqlCmd()
	var restoreCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "restore" {
			restoreCmd = sc
			break
		}
	}
	assert.NotNil(t, restoreCmd)

	assert.NotNil(t, restoreCmd.Flags().Lookup("input"), "restore 子命令缺少 input flag")

	dbFlags := []string{"host", "port", "user", "password", "database"}
	for _, f := range dbFlags {
		assert.NotNil(t, restoreCmd.Flags().Lookup(f), "restore 子命令缺少数据库连接 flag: %s", f)
	}
}

// ===== getPsqlDBFlags 测试 =====

func TestGetPsqlDBFlags_Defaults(t *testing.T) {
	cmd := NewPsqlCmd()
	// 获取主命令的 flags
	cmd.SetArgs([]string{})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	_ = cmd.Execute()

	flags := getPsqlDBFlags(cmd)
	assert.Equal(t, "127.0.0.1", flags.Host)
	assert.Equal(t, 5432, flags.Port)
	assert.Equal(t, "postgres", flags.User)
}
