package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// ===== NewMySQLCmd 测试 =====

func TestNewMySQLCmd_Basic(t *testing.T) {
	cmd := NewMySQLCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "mysql", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "MySQL")
}

func TestNewMySQLCmd_HasFlags(t *testing.T) {
	cmd := NewMySQLCmd()

	// 验证数据库通用参数已注册
	expectedFlags := []string{"host", "port", "user", "password", "database", "execute", "help"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "缺少 flag: %s", f)
	}
}

func TestNewMySQLCmd_FlagDefaults(t *testing.T) {
	cmd := NewMySQLCmd()

	host, _ := cmd.Flags().GetString("host")
	assert.Equal(t, "127.0.0.1", host, "host 默认值应为 127.0.0.1")

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 3306, port, "port 默认值应为 3306")

	user, _ := cmd.Flags().GetString("user")
	assert.Equal(t, "root", user, "user 默认值应为 root")

	password, _ := cmd.Flags().GetString("password")
	assert.Equal(t, "", password, "password 默认值应为空")

	database, _ := cmd.Flags().GetString("database")
	assert.Equal(t, "", database, "database 默认值应为空")

	execute, _ := cmd.Flags().GetString("execute")
	assert.Equal(t, "", execute, "execute 默认值应为空")
}

func TestNewMySQLCmd_PasswordNoOptDefVal(t *testing.T) {
	cmd := NewMySQLCmd()

	pwdFlag := cmd.Flags().Lookup("password")
	assert.NotNil(t, pwdFlag)
	assert.Equal(t, "ASK", pwdFlag.NoOptDefVal, "password flag 的 NoOptDefVal 应为 ASK")
}

func TestNewMySQLCmd_HasSubCommands(t *testing.T) {
	cmd := NewMySQLCmd()
	subCmds := cmd.Commands()

	subNames := make(map[string]bool)
	for _, sc := range subCmds {
		subNames[sc.Name()] = true
	}
	assert.True(t, subNames["dump"], "缺少 dump 子命令")
	assert.True(t, subNames["restore"], "缺少 restore 子命令")
}

func TestNewMySQLCmd_SilenceUsage(t *testing.T) {
	cmd := NewMySQLCmd()
	assert.True(t, cmd.SilenceUsage, "MySQL 命令应设置 SilenceUsage")
}

// ===== newMySQLDumpCmd 测试 =====

func TestNewMySQLDumpCmd_Basic(t *testing.T) {
	// 通过 MySQL 命令获取 dump 子命令
	cmd := NewMySQLCmd()
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

func TestNewMySQLDumpCmd_HasFlags(t *testing.T) {
	cmd := NewMySQLCmd()
	var dumpCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "dump" {
			dumpCmd = sc
			break
		}
	}
	assert.NotNil(t, dumpCmd)

	// 验证 dump 特有 flag
	dumpFlags := []string{"output", "format", "tables", "ignore-tables", "no-data", "no-schema"}
	for _, f := range dumpFlags {
		assert.NotNil(t, dumpCmd.Flags().Lookup(f), "dump 子命令缺少 flag: %s", f)
	}

	// 验证数据库连接参数也注册了
	dbFlags := []string{"host", "port", "user", "password", "database"}
	for _, f := range dbFlags {
		assert.NotNil(t, dumpCmd.Flags().Lookup(f), "dump 子命令缺少数据库连接 flag: %s", f)
	}
}

func TestNewMySQLDumpCmd_FlagDefaults(t *testing.T) {
	cmd := NewMySQLCmd()
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

// ===== newMySQLRestoreCmd 测试 =====

func TestNewMySQLRestoreCmd_Basic(t *testing.T) {
	cmd := NewMySQLCmd()
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

func TestNewMySQLRestoreCmd_HasFlags(t *testing.T) {
	cmd := NewMySQLCmd()
	var restoreCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "restore" {
			restoreCmd = sc
			break
		}
	}
	assert.NotNil(t, restoreCmd)

	// 验证 restore 特有 flag
	assert.NotNil(t, restoreCmd.Flags().Lookup("input"), "restore 子命令缺少 input flag")

	// 验证数据库连接参数也注册了
	dbFlags := []string{"host", "port", "user", "password", "database"}
	for _, f := range dbFlags {
		assert.NotNil(t, restoreCmd.Flags().Lookup(f), "restore 子命令缺少数据库连接 flag: %s", f)
	}
}

func TestNewMySQLRestoreCmd_FlagDefaults(t *testing.T) {
	cmd := NewMySQLCmd()
	var restoreCmd *cobra.Command
	for _, sc := range cmd.Commands() {
		if sc.Name() == "restore" {
			restoreCmd = sc
			break
		}
	}
	assert.NotNil(t, restoreCmd)

	input, _ := restoreCmd.Flags().GetString("input")
	assert.Equal(t, "", input, "input 默认值应为空")
}

// ===== addSubcommandDBFlags 测试 =====

func TestAddSubcommandDBFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addSubcommandDBFlags(cmd, DatabaseDefaults{
		Host: "localhost",
		Port: 3306,
		User: "admin",
	})

	// 验证所有 flag 已注册
	flags := []string{"host", "port", "user", "password", "database", "help"}
	for _, f := range flags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "addSubcommandDBFlags 应注册 flag: %s", f)
	}

	// 验证默认值
	host, _ := cmd.Flags().GetString("host")
	assert.Equal(t, "localhost", host)

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 3306, port)

	user, _ := cmd.Flags().GetString("user")
	assert.Equal(t, "admin", user)

	// 验证密码 NoOptDefVal
	pwdFlag := cmd.Flags().Lookup("password")
	assert.Equal(t, "ASK", pwdFlag.NoOptDefVal)
}

// ===== getSubcommandDBFlags 测试 =====

func TestGetSubcommandDBFlags_Defaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addSubcommandDBFlags(cmd, DatabaseDefaults{Host: "127.0.0.1", Port: 3306, User: "root"})

	cmd.SetArgs([]string{})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	_ = cmd.Execute()

	flags := getSubcommandDBFlags(cmd)
	assert.Equal(t, "127.0.0.1", flags.Host)
	assert.Equal(t, 3306, flags.Port)
	assert.Equal(t, "root", flags.User)
	assert.Equal(t, "", flags.Password)
	assert.Equal(t, "", flags.Database)
}
