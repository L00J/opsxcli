package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// NewConnectCmd 测试
// ============================================================================

// TestNewConnectCmd_命令属性 验证 connect 命令的基本属性
func TestNewConnectCmd_命令属性(t *testing.T) {
	cmd := NewConnectCmd()

	assert.Contains(t, cmd.Use, "connect")
	assert.Equal(t, "交互式SSH连接", cmd.Short)
	assert.NotNil(t, cmd.RunE)

	// 验证 flags 已注册
	assert.NotNil(t, cmd.Flags().Lookup("key"))
	assert.NotNil(t, cmd.Flags().Lookup("port"))
	assert.NotNil(t, cmd.Flags().Lookup("password"))

	// 验证默认值
	keyVal, _ := cmd.Flags().GetString("key")
	assert.Equal(t, "", keyVal, "默认密钥路径应为空")

	portVal, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 22, portVal, "默认端口应为22")

	passwordVal, _ := cmd.Flags().GetString("password")
	assert.Equal(t, "", passwordVal, "默认密码应为空")
}

// TestNewConnectCmd_参数不足 验证没有参数时报错
func TestNewConnectCmd_参数不足(t *testing.T) {
	cmd := NewConnectCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err, "缺少参数应该返回错误")
}

// TestNewConnectCmd_无效目标 验证无效目标地址时报错（不会真正连接）
func TestNewConnectCmd_无效目标(t *testing.T) {
	cmd := NewConnectCmd()
	// 多个@符号的目标会导致 parseTarget 报错
	cmd.SetArgs([]string{"a@b@c"})
	err := cmd.Execute()
	assert.Error(t, err, "多个@符号的目标应该返回错误")
}

// ============================================================================
// NewExecCmd 测试
// ============================================================================

// TestNewExecCmd_命令属性 验证 exec 命令的基本属性
func TestNewExecCmd_命令属性(t *testing.T) {
	cmd := NewExecCmd()

	assert.Contains(t, cmd.Use, "exec")
	assert.Equal(t, "在远程主机执行命令", cmd.Short)
	assert.NotNil(t, cmd.RunE)

	// 验证 flags 已注册
	assert.NotNil(t, cmd.Flags().Lookup("key"))
	assert.NotNil(t, cmd.Flags().Lookup("port"))
	assert.NotNil(t, cmd.Flags().Lookup("password"))
}

// TestNewExecCmd_参数不足 验证参数不足时报错
func TestNewExecCmd_参数不足(t *testing.T) {
	cmd := NewExecCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err, "缺少参数应该返回错误")
}

// TestNewExecCmd_仅目标无命令 验证只有目标没有命令时报错
func TestNewExecCmd_仅目标无命令(t *testing.T) {
	cmd := NewExecCmd()
	cmd.SetArgs([]string{"root@localhost"})
	err := cmd.Execute()
	assert.Error(t, err, "缺少命令参数应该返回错误")
}

// TestNewExecCmd_无效目标 验证无效目标地址时报错
func TestNewExecCmd_无效目标(t *testing.T) {
	cmd := NewExecCmd()
	cmd.SetArgs([]string{"a@b@c", "ls"})
	err := cmd.Execute()
	assert.Error(t, err, "多个@符号的目标应该返回错误")
}

// ============================================================================
// NewForwardCmd 测试
// ============================================================================

// TestNewForwardCmd_命令属性 验证 forward 命令的基本属性
func TestNewForwardCmd_命令属性(t *testing.T) {
	cmd := NewForwardCmd()

	assert.Contains(t, cmd.Use, "forward")
	assert.Equal(t, "SSH端口转发", cmd.Short)
	assert.NotNil(t, cmd.RunE)

	// 验证 flags 已注册
	assert.NotNil(t, cmd.Flags().Lookup("key"))
	assert.NotNil(t, cmd.Flags().Lookup("port"))
	assert.NotNil(t, cmd.Flags().Lookup("password"))
}

// TestNewForwardCmd_参数不足 验证参数不足时报错
func TestNewForwardCmd_参数不足(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err, "缺少参数应该返回错误")
}

// TestNewForwardCmd_未知模式 验证未知转发模式时报错
func TestNewForwardCmd_未知模式(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"unknown", "127.0.0.1:8080", "127.0.0.1:80", "root@localhost"})
	err := cmd.Execute()
	assert.Error(t, err, "未知转发模式应该返回错误")
	assert.Contains(t, err.Error(), "未知的转发模式")
}

// TestNewForwardCmd_Local参数不足 验证 local 模式参数不足时报错
func TestNewForwardCmd_Local参数不足(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"local", "127.0.0.1:8080"})
	err := cmd.Execute()
	assert.Error(t, err, "local模式参数不足应该返回错误")
	assert.Contains(t, err.Error(), "本地端口转发需要")
}

// TestNewForwardCmd_Remote参数不足 验证 remote 模式参数不足时报错
func TestNewForwardCmd_Remote参数不足(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"remote", "127.0.0.1:8080"})
	err := cmd.Execute()
	assert.Error(t, err, "remote模式参数不足应该返回错误")
	assert.Contains(t, err.Error(), "远程端口转发需要")
}

// TestNewForwardCmd_Dynamic参数不足 验证 dynamic 模式参数不足时报错
func TestNewForwardCmd_Dynamic参数不足(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"dynamic"})
	err := cmd.Execute()
	assert.Error(t, err, "dynamic模式参数不足应该返回错误")
	// Cobra MinimumNArgs 检查在自定义检查之前触发
	assert.Contains(t, err.Error(), "at least 2 arg")
}

// TestNewForwardCmd_Local无效绑定地址 验证 local 模式绑定地址无效时报错
func TestNewForwardCmd_Local无效绑定地址(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"local", "invalid", "127.0.0.1:80", "root@localhost"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的绑定地址应该返回错误")
}

// TestNewForwardCmd_Local无效目标地址 验证 local 模式目标地址无效时报错
func TestNewForwardCmd_Local无效目标地址(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"local", "127.0.0.1:8080", "invalid", "root@localhost"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的目标地址应该返回错误")
}

// TestNewForwardCmd_Local无效SSH目标 验证 local 模式SSH目标无效时报错
func TestNewForwardCmd_Local无效SSH目标(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"local", "127.0.0.1:8080", "127.0.0.1:80", "a@b@c"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的SSH目标应该返回错误")
}

// TestNewForwardCmd_Remote无效绑定地址 验证 remote 模式绑定地址无效时报错
func TestNewForwardCmd_Remote无效绑定地址(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"remote", "invalid", "127.0.0.1:80", "root@localhost"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的绑定地址应该返回错误")
}

// TestNewForwardCmd_Dynamic无效绑定地址 验证 dynamic 模式绑定地址无效时报错
func TestNewForwardCmd_Dynamic无效绑定地址(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"dynamic", "invalid", "root@localhost"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的绑定地址应该返回错误")
}

// TestNewForwardCmd_Dynamic无效SSH目标 验证 dynamic 模式SSH目标无效时报错
func TestNewForwardCmd_Dynamic无效SSH目标(t *testing.T) {
	cmd := NewForwardCmd()
	cmd.SetArgs([]string{"dynamic", "127.0.0.1:1080", "a@b@c"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的SSH目标应该返回错误")
}

// ============================================================================
// NewPutCmd 测试
// ============================================================================

// TestNewPutCmd_命令属性 验证 put 命令的基本属性
func TestNewPutCmd_命令属性(t *testing.T) {
	cmd := NewPutCmd()

	assert.Contains(t, cmd.Use, "put")
	assert.Equal(t, "上传文件或目录", cmd.Short)
	assert.NotNil(t, cmd.RunE)

	// 验证 flags 已注册
	assert.NotNil(t, cmd.Flags().Lookup("key"))
	assert.NotNil(t, cmd.Flags().Lookup("port"))
	assert.NotNil(t, cmd.Flags().Lookup("password"))
	assert.NotNil(t, cmd.Flags().Lookup("recursive"))
}

// TestNewPutCmd_参数不足 验证参数不足时报错
func TestNewPutCmd_参数不足(t *testing.T) {
	cmd := NewPutCmd()
	cmd.SetArgs([]string{"/tmp/file"})
	err := cmd.Execute()
	assert.Error(t, err, "缺少远程路径参数应该返回错误")
}

// TestNewPutCmd_参数过多 验证参数过多时报错
func TestNewPutCmd_参数过多(t *testing.T) {
	cmd := NewPutCmd()
	cmd.SetArgs([]string{"/tmp/file", "root@host:/tmp", "extra"})
	err := cmd.Execute()
	assert.Error(t, err, "参数过多应该返回错误")
}

// TestNewPutCmd_无效远程路径 验证远程路径无效时报错
func TestNewPutCmd_无效远程路径(t *testing.T) {
	cmd := NewPutCmd()
	// 远程路径缺少冒号
	cmd.SetArgs([]string{"/tmp/file", "invalid-no-colon"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的远程路径应该返回错误")
}

// TestNewPutCmd_多个At符号远程路径 验证远程路径包含多个@时报错
func TestNewPutCmd_多个At符号远程路径(t *testing.T) {
	cmd := NewPutCmd()
	cmd.SetArgs([]string{"/tmp/file", "a@b@c:/tmp/dest"})
	err := cmd.Execute()
	assert.Error(t, err, "远程路径包含多个@应该返回错误")
}

// ============================================================================
// NewGetCmd 测试
// ============================================================================

// TestNewGetCmd_命令属性 验证 get 命令的基本属性
func TestNewGetCmd_命令属性(t *testing.T) {
	cmd := NewGetCmd()

	assert.Contains(t, cmd.Use, "get")
	assert.Equal(t, "下载文件或目录", cmd.Short)
	assert.NotNil(t, cmd.RunE)

	// 验证 flags 已注册
	assert.NotNil(t, cmd.Flags().Lookup("key"))
	assert.NotNil(t, cmd.Flags().Lookup("port"))
	assert.NotNil(t, cmd.Flags().Lookup("password"))
	assert.NotNil(t, cmd.Flags().Lookup("recursive"))
}

// TestNewGetCmd_参数不足 验证参数不足时报错
func TestNewGetCmd_参数不足(t *testing.T) {
	cmd := NewGetCmd()
	cmd.SetArgs([]string{"root@host:/tmp/file"})
	err := cmd.Execute()
	assert.Error(t, err, "缺少本地路径参数应该返回错误")
}

// TestNewGetCmd_无效远程路径 验证远程路径无效时报错
func TestNewGetCmd_无效远程路径(t *testing.T) {
	cmd := NewGetCmd()
	cmd.SetArgs([]string{"invalid-no-colon", "/tmp/local"})
	err := cmd.Execute()
	assert.Error(t, err, "无效的远程路径应该返回错误")
}

// ============================================================================
// Connect 导出函数测试
// ============================================================================

// TestConnect_无效目标 验证 Connect 导出函数对无效目标的处理
func TestConnect_无效目标(t *testing.T) {
	err := Connect("a@b@c", "", 22, "")
	assert.Error(t, err, "多个@符号的目标应该返回错误")
}

// TestConnect_空目标 验证 Connect 对空目标的处理
func TestConnect_空目标(t *testing.T) {
	// 空字符串 parseTarget 不会报错（默认 root 用户），但 NewSSHClient 会失败
	err := Connect("", "", 22, "")
	// 会因为空主机名无法连接而报错
	assert.Error(t, err)
}

// ============================================================================
// ExecCommand 导出函数测试
// ============================================================================

// TestExecCommand_无效目标 验证 ExecCommand 导出函数对无效目标的处理
func TestExecCommand_无效目标(t *testing.T) {
	err := ExecCommand("a@b@c", "", 22, "", "ls")
	assert.Error(t, err, "多个@符号的目标应该返回错误")
}

// TestExecCommand_有效目标不可达 验证有效目标但不可达时的行为
func TestExecCommand_有效目标不可达(t *testing.T) {
	// 使用 RFC 5737 测试地址，不会真正连接
	err := ExecCommand("root@192.0.2.1", "", 22, "", "ls")
	assert.Error(t, err, "不可达地址应该返回错误")
}

// ============================================================================
// forwardLocal/forwardRemote/forwardDynamic 直接测试
// ============================================================================

// TestForwardLocal_无效绑定地址 验证 forwardLocal 对无效绑定地址的处理
func TestForwardLocal_无效绑定地址(t *testing.T) {
	err := forwardLocal("invalid-no-colon", "127.0.0.1:80", "root@localhost", "", 22, "")
	assert.Error(t, err, "无效的绑定地址应该返回错误")
}

// TestForwardLocal_无效目标地址 验证 forwardLocal 对无效目标地址的处理
func TestForwardLocal_无效目标地址(t *testing.T) {
	err := forwardLocal("127.0.0.1:8080", "invalid", "root@localhost", "", 22, "")
	assert.Error(t, err, "无效的目标地址应该返回错误")
}

// TestForwardLocal_无效SSH目标 验证 forwardLocal 对无效SSH目标的处理
func TestForwardLocal_无效SSH目标(t *testing.T) {
	err := forwardLocal("127.0.0.1:8080", "127.0.0.1:80", "a@b@c", "", 22, "")
	assert.Error(t, err, "无效的SSH目标应该返回错误")
}

// TestForwardRemote_无效绑定地址 验证 forwardRemote 对无效绑定地址的处理
func TestForwardRemote_无效绑定地址(t *testing.T) {
	err := forwardRemote("invalid", "127.0.0.1:80", "root@localhost", "", 22, "")
	assert.Error(t, err, "无效的绑定地址应该返回错误")
}

// TestForwardRemote_无效目标地址 验证 forwardRemote 对无效目标地址的处理
func TestForwardRemote_无效目标地址(t *testing.T) {
	err := forwardRemote("127.0.0.1:8080", "invalid", "root@localhost", "", 22, "")
	assert.Error(t, err, "无效的目标地址应该返回错误")
}

// TestForwardRemote_无效SSH目标 验证 forwardRemote 对无效SSH目标的处理
func TestForwardRemote_无效SSH目标(t *testing.T) {
	err := forwardRemote("127.0.0.1:8080", "127.0.0.1:80", "a@b@c", "", 22, "")
	assert.Error(t, err, "无效的SSH目标应该返回错误")
}

// TestForwardDynamic_无效绑定地址 验证 forwardDynamic 对无效绑定地址的处理
func TestForwardDynamic_无效绑定地址(t *testing.T) {
	err := forwardDynamic("invalid", "root@localhost", "", 22, "")
	assert.Error(t, err, "无效的绑定地址应该返回错误")
}

// TestForwardDynamic_无效SSH目标 验证 forwardDynamic 对无效SSH目标的处理
func TestForwardDynamic_无效SSH目标(t *testing.T) {
	err := forwardDynamic("127.0.0.1:1080", "a@b@c", "", 22, "")
	assert.Error(t, err, "无效的SSH目标应该返回错误")
}

// ============================================================================
// putFile/getFile 直接测试
// ============================================================================

// TestPutFile_无效远程路径 验证 putFile 对无效远程路径的处理
func TestPutFile_无效远程路径(t *testing.T) {
	err := putFile("/tmp/local", "invalid-no-colon", "", 22, "", false)
	assert.Error(t, err, "无效的远程路径应该返回错误")
}

// TestPutFile_多个At符号 验证 putFile 对多个@符号的处理
func TestPutFile_多个At符号(t *testing.T) {
	err := putFile("/tmp/local", "a@b@c:/remote", "", 22, "", false)
	assert.Error(t, err, "多个@符号应该返回错误")
}

// TestGetFile_无效远程路径 验证 getFile 对无效远程路径的处理
func TestGetFile_无效远程路径(t *testing.T) {
	err := getFile("invalid-no-colon", "/tmp/local", "", 22, "", false)
	assert.Error(t, err, "无效的远程路径应该返回错误")
}

// TestGetFile_多个At符号 验证 getFile 对多个@符号的处理
func TestGetFile_多个At符号(t *testing.T) {
	err := getFile("a@b@c:/remote", "/tmp/local", "", 22, "", false)
	assert.Error(t, err, "多个@符号应该返回错误")
}
