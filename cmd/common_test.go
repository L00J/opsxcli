package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// ===== SetDatabasePassword / GetDatabasePassword / FetchAndClearDatabasePassword 测试 =====

func TestSetAndGetDatabasePassword(t *testing.T) {
	// 清理状态
	securePassword = nil

	// 设置密码
	SetDatabasePassword("mypassword")

	// 获取密码
	pwd := GetDatabasePassword()
	assert.Equal(t, "mypassword", pwd)

	// 再次获取仍然可用（Get 不会清零）
	pwd = GetDatabasePassword()
	assert.Equal(t, "mypassword", pwd)
}

func TestGetDatabasePassword_Empty(t *testing.T) {
	securePassword = nil
	pwd := GetDatabasePassword()
	assert.Equal(t, "", pwd)
}

func TestFetchAndClearDatabasePassword(t *testing.T) {
	securePassword = nil

	SetDatabasePassword("secretpwd")

	// FetchAndClear 获取后清零
	pwd := FetchAndClearDatabasePassword()
	assert.Equal(t, "secretpwd", pwd)

	// 清零后再次获取应为空
	pwd = FetchAndClearDatabasePassword()
	assert.Equal(t, "", pwd)
}

func TestFetchAndClearDatabasePassword_Empty(t *testing.T) {
	securePassword = nil

	pwd := FetchAndClearDatabasePassword()
	assert.Equal(t, "", pwd)
}

func TestFetchAndClearDatabasePassword_MemoryZeroed(t *testing.T) {
	securePassword = nil

	SetDatabasePassword("test123")
	// 手动检查 securePassword 已设置
	assert.NotNil(t, securePassword)

	// FetchAndClear 应清零内存
	_ = FetchAndClearDatabasePassword()
	assert.Nil(t, securePassword)
}

// ===== DatabaseFlags 测试 =====

func TestDatabaseFlags_Struct(t *testing.T) {
	flags := DatabaseFlags{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "testdb",
		Execute:  "SELECT 1",
	}
	assert.Equal(t, "localhost", flags.Host)
	assert.Equal(t, 3306, flags.Port)
	assert.Equal(t, "root", flags.User)
	assert.Equal(t, "secret", flags.Password)
	assert.Equal(t, "testdb", flags.Database)
	assert.Equal(t, "SELECT 1", flags.Execute)
}

func TestAddDatabaseFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	defaults := DatabaseDefaults{
		Host: "127.0.0.1",
		Port: 3306,
		User: "root",
	}
	AddDatabaseFlags(cmd, defaults)

	// 验证所有 flag 已注册
	assert.NotNil(t, cmd.Flags().Lookup("host"))
	assert.NotNil(t, cmd.Flags().Lookup("port"))
	assert.NotNil(t, cmd.Flags().Lookup("user"))
	assert.NotNil(t, cmd.Flags().Lookup("password"))
	assert.NotNil(t, cmd.Flags().Lookup("database"))
	assert.NotNil(t, cmd.Flags().Lookup("execute"))
	assert.NotNil(t, cmd.Flags().Lookup("help"))

	// 验证默认值
	host, _ := cmd.Flags().GetString("host")
	assert.Equal(t, "127.0.0.1", host)

	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 3306, port)

	user, _ := cmd.Flags().GetString("user")
	assert.Equal(t, "root", user)

	// 验证密码 NoOptDefVal
	pwdFlag := cmd.Flags().Lookup("password")
	assert.Equal(t, "ASK", pwdFlag.NoOptDefVal)
}

func TestGetDatabaseFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	defaults := DatabaseDefaults{Host: "localhost", Port: 5432, User: "postgres"}
	AddDatabaseFlags(cmd, defaults)

	// 设置 flag 值（NoOptDefVal 设置后，需要用 --flag=value 形式避免歧义）
	cmd.SetArgs([]string{"--host=db.example.com", "--port=5433", "--user=admin", "--password=mypass", "--database=mydb", "--execute=SELECT 1"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	_ = cmd.Execute()

	flags := GetDatabaseFlags(cmd)
	assert.Equal(t, "db.example.com", flags.Host)
	assert.Equal(t, 5433, flags.Port)
	assert.Equal(t, "admin", flags.User)
	assert.Equal(t, "mypass", flags.Password)
	assert.Equal(t, "mydb", flags.Database)
	assert.Equal(t, "SELECT 1", flags.Execute)
}

func TestGetDatabaseFlags_ASKPassword(t *testing.T) {
	// 清理状态
	securePassword = nil

	cmd := &cobra.Command{Use: "test"}
	defaults := DatabaseDefaults{Host: "localhost", Port: 3306, User: "root"}
	AddDatabaseFlags(cmd, defaults)

	// 模拟 -p 不带参数（NoOptDefVal = "ASK"）
	cmd.SetArgs([]string{"-p"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }

	// 预先通过 SetDatabasePassword 设置密码（模拟 main.go 的参数预处理）
	SetDatabasePassword("preprocessed_pwd")

	_ = cmd.Execute()

	flags := GetDatabaseFlags(cmd)
	// 当 password=="ASK" 时，应使用 securePassword 中的值
	assert.Equal(t, "preprocessed_pwd", flags.Password)

	// 密码已被 FetchAndClear 清零
	assert.Nil(t, securePassword)
}

func TestGetDatabaseFlags_ASKPassword_Empty(t *testing.T) {
	securePassword = nil

	cmd := &cobra.Command{Use: "test"}
	defaults := DatabaseDefaults{Host: "localhost", Port: 3306, User: "root"}
	AddDatabaseFlags(cmd, defaults)

	cmd.SetArgs([]string{"-p"})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	_ = cmd.Execute()

	flags := GetDatabaseFlags(cmd)
	// 没有 securePassword，密码保持 ASK（后续会触发交互式输入）
	assert.Equal(t, "ASK", flags.Password)
}

func TestGetDatabaseFlags_Defaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	defaults := DatabaseDefaults{Host: "127.0.0.1", Port: 3306, User: "root"}
	AddDatabaseFlags(cmd, defaults)

	cmd.SetArgs([]string{})
	cmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	_ = cmd.Execute()

	flags := GetDatabaseFlags(cmd)
	assert.Equal(t, "127.0.0.1", flags.Host)
	assert.Equal(t, 3306, flags.Port)
	assert.Equal(t, "root", flags.User)
	assert.Equal(t, "", flags.Password)
	assert.Equal(t, "", flags.Database)
	assert.Equal(t, "", flags.Execute)
}

// ===== DatabaseDefaults 测试 =====

func TestDatabaseDefaults_Struct(t *testing.T) {
	defaults := DatabaseDefaults{
		Host: "localhost",
		Port: 5432,
		User: "admin",
	}
	assert.Equal(t, "localhost", defaults.Host)
	assert.Equal(t, 5432, defaults.Port)
	assert.Equal(t, "admin", defaults.User)
}
