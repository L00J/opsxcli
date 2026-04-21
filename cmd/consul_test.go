package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== NewConsulCmd 测试 =====

func TestNewConsulCmd_Basic(t *testing.T) {
	cmd := NewConsulCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "consul")
	assert.NotEmpty(t, cmd.Short)
	assert.Contains(t, cmd.Short, "Consul")
}

func TestNewConsulCmd_Long(t *testing.T) {
	cmd := NewConsulCmd()
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "Kubernetes")
}

func TestNewConsulCmd_SilenceUsage(t *testing.T) {
	cmd := NewConsulCmd()
	assert.True(t, cmd.SilenceUsage, "consul 命令应设置 SilenceUsage")
}

func TestNewConsulCmd_HasRunE(t *testing.T) {
	cmd := NewConsulCmd()
	assert.NotNil(t, cmd.RunE, "consul 命令应有 RunE")
}

func TestNewConsulCmd_HasFlags(t *testing.T) {
	cmd := NewConsulCmd()

	commonFlags := []string{"service", "metrics", "clean"}
	k8sFlags := []string{"kubeconfig", "clear-cache"}
	cloudFlags := []string{"hosts", "hosts-file", "app-port", "node-exp-port", "skip-node-exporter"}

	for _, f := range commonFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "consul 缺少通用 flag: %s", f)
	}
	for _, f := range k8sFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "consul 缺少 K8s flag: %s", f)
	}
	for _, f := range cloudFlags {
		assert.NotNil(t, cmd.Flags().Lookup(f), "consul 缺少云主机 flag: %s", f)
	}
}

func TestNewConsulCmd_FlagDefaults(t *testing.T) {
	cmd := NewConsulCmd()

	service, _ := cmd.Flags().GetString("service")
	assert.Equal(t, "", service, "service 默认值应为空")

	metrics, _ := cmd.Flags().GetString("metrics")
	assert.Equal(t, "/actuator/prometheus", metrics, "metrics 默认值应为 /actuator/prometheus")

	clean, _ := cmd.Flags().GetBool("clean")
	assert.False(t, clean, "clean 默认值应为 false")

	kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
	assert.Equal(t, "", kubeconfig, "kubeconfig 默认值应为空")

	clearCache, _ := cmd.Flags().GetBool("clear-cache")
	assert.False(t, clearCache, "clear-cache 默认值应为 false")

	hosts, _ := cmd.Flags().GetStringSlice("hosts")
	assert.Empty(t, hosts, "hosts 默认值应为空")

	hostsFile, _ := cmd.Flags().GetString("hosts-file")
	assert.Equal(t, "", hostsFile, "hosts-file 默认值应为空")

	appPort, _ := cmd.Flags().GetInt("app-port")
	assert.Equal(t, 8080, appPort, "app-port 默认值应为 8080")

	nodeExpPort, _ := cmd.Flags().GetInt("node-exp-port")
	assert.Equal(t, 9100, nodeExpPort, "node-exp-port 默认值应为 9100")

	skipNodeExporter, _ := cmd.Flags().GetBool("skip-node-exporter")
	assert.False(t, skipNodeExporter, "skip-node-exporter 默认值应为 false")
}

func TestNewConsulCmd_FlagShorthands(t *testing.T) {
	cmd := NewConsulCmd()
	assert.Equal(t, "s", cmd.Flags().Lookup("service").Shorthand)
	assert.Equal(t, "m", cmd.Flags().Lookup("metrics").Shorthand)
	assert.Equal(t, "k", cmd.Flags().Lookup("kubeconfig").Shorthand)
}

func TestNewConsulCmd_RequiresService(t *testing.T) {
	cmd := NewConsulCmd()
	// service is marked required
	assert.NotNil(t, cmd.Flags().Lookup("service"))
}
