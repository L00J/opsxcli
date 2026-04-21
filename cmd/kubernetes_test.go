package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewKubernetesCmd 测试 =====

func TestNewKubernetesCmd_Basic(t *testing.T) {
	cmd := NewKubernetesCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "kubernetes", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "Kubernetes")
}

func TestNewKubernetesCmd_HasSubCommands(t *testing.T) {
	cmd := NewKubernetesCmd()
	subCmds := cmd.Commands()

	// 验证有子命令（check, resource, yaml）
	assert.True(t, len(subCmds) >= 3, "Kubernetes 命令应至少有 check, resource, yaml 子命令")

	subNames := make(map[string]bool)
	for _, sc := range subCmds {
		subNames[sc.Name()] = true
	}
	assert.True(t, subNames["check"], "缺少 check 子命令")
	assert.True(t, subNames["resource"], "缺少 resource 子命令")
	assert.True(t, subNames["yaml"], "缺少 yaml 子命令")
}

func TestNewKubernetesCmd_DefaultRunShowsHelp(t *testing.T) {
	cmd := NewKubernetesCmd()
	// 主命令的 Run 应调用 Help
	assert.NotNil(t, cmd.Run, "Kubernetes 主命令应有 Run 函数（显示帮助）")
}

// ===== NewKubernetesCheckCmd 测试 =====

func TestNewKubernetesCheckCmd_Basic(t *testing.T) {
	cmd := NewKubernetesCheckCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "check", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "健康检查")
}

func TestNewKubernetesCheckCmd_HasFlags(t *testing.T) {
	cmd := NewKubernetesCheckCmd()

	assert.NotNil(t, cmd.Flags().Lookup("kubeconfig"), "缺少 kubeconfig flag")
}

func TestNewKubernetesCheckCmd_FlagDefaults(t *testing.T) {
	cmd := NewKubernetesCheckCmd()

	kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
	assert.Equal(t, "", kubeconfig, "kubeconfig 默认值应为空（使用默认 ~/.kube/config）")
}

func TestNewKubernetesCheckCmd_FlagShorthand(t *testing.T) {
	cmd := NewKubernetesCheckCmd()

	assert.Equal(t, "k", cmd.Flags().Lookup("kubeconfig").Shorthand, "kubeconfig 的 shorthand 应为 k")
}
