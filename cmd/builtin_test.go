package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Builtin Disk 命令测试 =====

func TestNewDdCmd_Basic(t *testing.T) {
	cmd := NewDdCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
	assert.True(t, cmd.DisableFlagParsing, "dd 应禁用标准参数解析")
}

func TestNewDdCmd_HelpFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"-h 帮助", []string{"-h"}},
		{"--help 帮助", []string{"--help"}},
		{"help 子命令", []string{"help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewDdCmd()
			cmd.SetArgs(tt.args)

			// dd 命令帮助输出到 os.Stdout（fmt.Println），不通过 SetOut
			// 只需验证执行不报错即可
			err := cmd.Execute()
			require.NoError(t, err)
		})
	}
}

func TestNewDdCmd_NoArgs(t *testing.T) {
	cmd := NewDdCmd()
	cmd.SetArgs([]string{})

	// dd 无参数时 builtin.ParseDdArgs 返回空选项
	// 实际行为取决于 builtin.Dd 的实现，可能不报错
	err := cmd.Execute()
	// 不强制要求错误，验证不 panic 即可
	_ = err
}

// ===== Builtin File 命令测试 =====

func TestNewLsCmd_Basic(t *testing.T) {
	cmd := NewLsCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewCpCmd_Basic(t *testing.T) {
	cmd := NewCpCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewMvCmd_Basic(t *testing.T) {
	cmd := NewMvCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewRmCmd_Basic(t *testing.T) {
	cmd := NewRmCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewMkdirCmd_Basic(t *testing.T) {
	cmd := NewMkdirCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewRmdirCmd_Basic(t *testing.T) {
	cmd := NewRmdirCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewTouchCmd_Basic(t *testing.T) {
	cmd := NewTouchCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewChmodCmd_Basic(t *testing.T) {
	cmd := NewChmodCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewChownCmd_Basic(t *testing.T) {
	cmd := NewChownCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewLnCmd_Basic(t *testing.T) {
	cmd := NewLnCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewCatCmd_Basic(t *testing.T) {
	cmd := NewCatCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Builtin Network 命令测试 =====

func TestNewIfconfigCmd_Basic(t *testing.T) {
	cmd := NewIfconfigCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewRouteCmd_Basic(t *testing.T) {
	cmd := NewRouteCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewIpCmd_Basic(t *testing.T) {
	cmd := NewIpCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Builtin Text 命令测试 =====

func TestNewHeadCmd_Basic(t *testing.T) {
	cmd := NewHeadCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewTailCmd_Basic(t *testing.T) {
	cmd := NewTailCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewGrepCmd_Basic(t *testing.T) {
	cmd := NewGrepCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}
