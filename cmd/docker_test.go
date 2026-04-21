package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewDockerCmd 测试 =====

func TestNewDockerCmd_Basic(t *testing.T) {
	cmd := NewDockerCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "docker", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "Docker")
}

func TestNewDockerCmd_HasRun(t *testing.T) {
	cmd := NewDockerCmd()
	assert.NotNil(t, cmd.Run, "docker 主命令应有 Run（显示帮助）")
}

func TestNewDockerCmd_HasPullSubCommand(t *testing.T) {
	cmd := NewDockerCmd()
	subCmds := cmd.Commands()

	subNames := make(map[string]bool)
	for _, sc := range subCmds {
		subNames[sc.Name()] = true
	}
	assert.True(t, subNames["pull"], "docker 命令应有 pull 子命令")
}

// ===== NewDockerPullCmd 测试 =====

func TestNewDockerPullCmd_Basic(t *testing.T) {
	cmd := NewDockerPullCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "pull")
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "拉取")
}

func TestNewDockerPullCmd_HasFlags(t *testing.T) {
	cmd := NewDockerPullCmd()

	expectedFlags := []string{"registry", "concurrency"}
	for _, f := range expectedFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "docker pull 缺少 flag: %s", f)
	}
}

func TestNewDockerPullCmd_FlagDefaults(t *testing.T) {
	cmd := NewDockerPullCmd()

	concurrency, _ := cmd.Flags().GetInt("concurrency")
	assert.Equal(t, 5, concurrency, "concurrency 默认值应为 5")

	registries, _ := cmd.Flags().GetStringSlice("registry")
	assert.Empty(t, registries, "registry 默认值应为空")
}

func TestNewDockerPullCmd_FlagShorthands(t *testing.T) {
	cmd := NewDockerPullCmd()

	assert.Equal(t, "r", cmd.Flags().Lookup("registry").Shorthand)
	assert.Equal(t, "c", cmd.Flags().Lookup("concurrency").Shorthand)
}

func TestNewDockerPullCmd_RequiresArgs(t *testing.T) {
	cmd := NewDockerPullCmd()
	// 验证 MinimumNArgs(1)
	err := cmd.Execute()
	assert.Error(t, err, "docker pull 无参数应报错")
}

func TestNewDockerPullCmd_SilenceUsage(t *testing.T) {
	cmd := NewDockerPullCmd()
	assert.True(t, cmd.SilenceUsage, "docker pull 应设置 SilenceUsage")
}
