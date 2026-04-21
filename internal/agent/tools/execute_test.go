package tools

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===================== UnifiedExecuteTool 元数据测试 =====================

func TestUnifiedExecuteTool_Metadata(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	assert.Equal(t, "execute", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotNil(t, tool.Parameters())
	assert.Equal(t, RiskMedium, tool.RiskLevel())

	// 验证参数定义包含必要字段
	params := tool.Parameters()
	props := params["properties"].(map[string]interface{})
	assert.Contains(t, props, "command")
	assert.Contains(t, props, "host")
	assert.Contains(t, props, "timeout")
	assert.Contains(t, props, "working_dir")
	assert.Contains(t, props, "use_sudo")
}

func TestUnifiedExecuteTool_ParametersRequired(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	params := tool.Parameters()
	required := params["required"].([]string)
	assert.Contains(t, required, "command")
}

// ===================== UnifiedExecuteTool 本地执行测试 =====================

func TestUnifiedExecuteTool_LocalEcho(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo hello_opsxcli",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "hello_opsxcli")
	// 本地执行的 Summary 应有 [本地] 前缀
	assert.Contains(t, result.Summary, "[本地]")
}

func TestUnifiedExecuteTool_LocalEmptyCommand(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "",
	})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "command 参数不能为空")
}

func TestUnifiedExecuteTool_LocalMissingCommand(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestUnifiedExecuteTool_LocalUname(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "uname -s",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestUnifiedExecuteTool_LocalWithWorkingDir(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command":     "pwd",
		"working_dir": "/tmp",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	// 不同系统 /tmp 可能是软链接，只检查包含 tmp
	assert.Contains(t, result.Output, "tmp")
}

func TestUnifiedExecuteTool_LocalTimeout(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "sleep 10",
		"timeout": 0.5,
	})
	// 超时不返回 error，但 result.Success=false
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestUnifiedExecuteTool_LocalDangerousCommand(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "rm -rf /",
	})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

// ===================== UnifiedExecuteTool 远程执行测试 =====================
// 远程执行需要 SSH 服务，仅测试参数验证和路由逻辑

func TestUnifiedExecuteTool_RemoteMissingHost(t *testing.T) {
	// 不指定 host 时应路由到本地执行
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo local_test",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "local_test")
}

func TestUnifiedExecuteTool_RemoteInvalidHost(t *testing.T) {
	// 指定无效 host 应该连接失败
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo test",
		"host":    "nonexistent@invalid.host.that.does.not.exist:22",
	})
	// 连接失败应该返回错误
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotNil(t, result)
		assert.False(t, result.Success)
	}
}

// ===================== UnifiedExecuteTool Closer 接口测试 =====================

func TestUnifiedExecuteTool_Close(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	// Close 应该不 panic
	assert.NoError(t, tool.Close())
	// 多次 Close 也应该安全
	assert.NoError(t, tool.Close())
}

func TestUnifiedExecuteTool_ImplementsCloser(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	// 验证实现了 Closer 接口
	var _ Closer = tool
}

// ===================== UnifiedExecuteTool 集成测试 =====================

func TestUnifiedExecuteTool_EnvironmentVariables(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo $HOME",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Output)
}

func TestUnifiedExecuteTool_PipeCommands(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo 'hello world' | awk '{print $1}'",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "hello")
}

func TestUnifiedExecuteTool_MultipleCommands(t *testing.T) {
	tool := NewUnifiedExecuteTool()
	defer tool.Close()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo first && echo second",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "first")
	assert.Contains(t, result.Output, "second")
}

// ===================== registry 集成测试 =====================

func TestRegistry_GetUnifiedTools(t *testing.T) {
	r := NewRegistry()
	r.RegisterDefaults()

	// execute 工具应该可以获取
	exeTool, err := r.Get("execute")
	assert.NoError(t, err)
	assert.Equal(t, "execute", exeTool.Name())

	// transfer 工具应该可以获取
	transferTool, err := r.Get("transfer")
	assert.NoError(t, err)
	assert.Equal(t, "transfer", transferTool.Name())

	// 旧工具仍然可用
	localTool, err := r.Get("local_bash")
	assert.NoError(t, err)
	assert.Equal(t, "local_bash", localTool.Name())
}

// ===================== UnifiedTransferTool 测试 =====================

func TestUnifiedTransferTool_Metadata(t *testing.T) {
	tool := NewUnifiedTransferTool()
	assert.Equal(t, "transfer", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotNil(t, tool.Parameters())
	assert.Equal(t, RiskMedium, tool.RiskLevel())
}

func TestUnifiedTransferTool_ParametersRequired(t *testing.T) {
	tool := NewUnifiedTransferTool()

	params := tool.Parameters()
	required := params["required"].([]string)
	assert.Contains(t, required, "source")
	assert.Contains(t, required, "destination")
}

func TestUnifiedTransferTool_MissingSource(t *testing.T) {
	tool := NewUnifiedTransferTool()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"destination": "/tmp/dest.txt",
	})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "source 参数不能为空")
}

func TestUnifiedTransferTool_MissingDestination(t *testing.T) {
	tool := NewUnifiedTransferTool()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source": "/tmp/src.txt",
	})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "destination 参数不能为空")
}

func TestUnifiedTransferTool_LocalFileCopy(t *testing.T) {
	// 创建临时源文件
	srcFile, err := os.CreateTemp("", "opsxcli_test_src_*.txt")
	assert.NoError(t, err)
	srcPath := srcFile.Name()
	defer os.Remove(srcPath)
	srcFile.WriteString("test content for opsxcli transfer")

	dstPath := os.TempDir() + "/opsxcli_test_dst.txt"
	defer os.Remove(dstPath)

	tool := NewUnifiedTransferTool()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source":      srcPath,
		"destination": dstPath,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "已复制")

	// 验证文件内容
	content, err := os.ReadFile(dstPath)
	assert.NoError(t, err)
	assert.Equal(t, "test content for opsxcli transfer", string(content))
}

func TestUnifiedTransferTool_LocalDirectoryCopy(t *testing.T) {
	// 创建临时源目录
	srcDir, err := os.MkdirTemp("", "opsxcli_test_srcdir_*")
	assert.NoError(t, err)
	defer os.RemoveAll(srcDir)

	// 创建测试文件
	os.WriteFile(srcDir+"/file1.txt", []byte("hello"), 0644)
	os.WriteFile(srcDir+"/file2.txt", []byte("world"), 0644)

	dstDir := os.TempDir() + "/opsxcli_test_dstdir"
	defer os.RemoveAll(dstDir)

	tool := NewUnifiedTransferTool()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source":      srcDir,
		"destination": dstDir,
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "已复制目录")

	// 验证文件存在
	_, err = os.Stat(dstDir + "/file1.txt")
	assert.NoError(t, err)
	_, err = os.Stat(dstDir + "/file2.txt")
	assert.NoError(t, err)
}

func TestUnifiedTransferTool_LocalNonexistentSource(t *testing.T) {
	tool := NewUnifiedTransferTool()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source":      "/nonexistent/path/file.txt",
		"destination": "/tmp/dest.txt",
	})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestUnifiedTransferTool_RemoteMissingDirection(t *testing.T) {
	tool := NewUnifiedTransferTool()

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"source":      "/tmp/src.txt",
		"destination": "/tmp/dst.txt",
		"host":        "root@invalid.host",
	})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "direction")
}

func TestUnifiedTransferTool_CancelledContext(t *testing.T) {
	tool := NewUnifiedTransferTool()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	srcFile, _ := os.CreateTemp("", "opsxcli_cancel_test_*.txt")
	defer os.Remove(srcFile.Name())
	srcFile.WriteString("data")

	result, err := tool.Execute(ctx, map[string]interface{}{
		"source":      srcFile.Name(),
		"destination": os.TempDir() + "/opsxcli_cancel_dst.txt",
	})
	// 取消后应该返回错误
	if err != nil {
		assert.Error(t, err)
	} else if result != nil {
		assert.False(t, result.Success)
	}
}
