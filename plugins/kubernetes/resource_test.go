package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========== resource.go: extractDeploymentName 边界用例 ==========

func TestExtractDeploymentName_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		podName  string
		expected string
	}{
		{"DaemonSet三段名", "kube-proxy-abcdef", "kube-proxy"},
		{"CronJob Pod名", "my-cron-1234567890-abcde", "my-cron"},
		{"Job带短哈希", "backup-abc12", "backup"},
		{"纯数字后缀", "app-123", "app"},
		{"两段名无哈希", "myapp-abc", "myapp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDeploymentName(tt.podName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ========== resource.go: parseCPUValue 边界用例 ==========

func TestParseCPUValue_Invalid(t *testing.T) {
	// 无效格式应该返回0
	result := parseCPUValue("abc")
	assert.Equal(t, 0.0, result)
}

func TestParseCPUValue_LargeMillis(t *testing.T) {
	result := parseCPUValue("4000m")
	assert.InDelta(t, 4.0, result, 0.001)
}

// ========== resource.go: parseMemoryValue 边界用例 ==========

func TestParseMemoryValue_Bytes(t *testing.T) {
	// 纯字节数 → MB
	result := parseMemoryValue("1048576") // 1MB in bytes
	assert.InDelta(t, 1.0, result, 0.01)
}

func TestParseMemoryValue_Invalid(t *testing.T) {
	result := parseMemoryValue("abc")
	assert.InDelta(t, 0.0, result, 0.01)
}

// ========== resource.go: formatCPUValue 边界用例 ==========

func TestFormatCPUValue_ExactIntegerMillis(t *testing.T) {
	// 0.25 cores = 250m (整数毫核)
	result := formatCPUValue(0.25)
	assert.Equal(t, "250m", result)
}

func TestFormatCPUValue_VerySmallValue(t *testing.T) {
	// 0.01 cores = 10m
	result := formatCPUValue(0.01)
	assert.Equal(t, "10m", result)
}

// ========== resource.go: formatMemoryValue 边界用例 ==========

func TestFormatMemoryValue_ExactlyOneGB(t *testing.T) {
	result := formatMemoryValue(1024)
	assert.Equal(t, "1.00 Gi", result)
}

// ========== resource.go: isAlphaNumeric 边界用例 ==========

func TestIsAlphaNumeric_EdgeCases(t *testing.T) {
	assert.False(t, isAlphaNumeric("abc.def"))
	assert.True(t, isAlphaNumeric("ABCDEF123456"))
	assert.False(t, isAlphaNumeric("hello world"))
}

// ========== resource.go: hasUpperCase 边界用例 ==========

func TestHasUpperCase_AllDigits(t *testing.T) {
	assert.False(t, hasUpperCase("123456"))
}

// ========== DeploymentResource 结构体测试 ==========

func TestDeploymentResource_Struct(t *testing.T) {
	d := DeploymentResource{
		Namespace:           "production",
		DeploymentName:      "api-server",
		Replicas:            3,
		AvailableReplicas:   3,
		CapacityProvisioned: "2vCPU 4GB",
		CPURequest:          2.0,
		CPULimit:            4.0,
		MemoryRequest:       4096.0,
		MemoryLimit:         8192.0,
	}
	assert.Equal(t, "production", d.Namespace)
	assert.Equal(t, "api-server", d.DeploymentName)
	assert.Equal(t, int32(3), d.Replicas)
	assert.Equal(t, int32(3), d.AvailableReplicas)
	assert.Equal(t, 2.0, d.CPURequest)
	assert.Equal(t, 8192.0, d.MemoryLimit)
}

// ========== systemNamespaces 测试 ==========

func TestSystemNamespaces_Completeness(t *testing.T) {
	knownSystems := []string{
		"kube-system", "ingress-nginx", "prometheus",
		"istio-system", "devops", "monitor", "logging",
	}
	for _, ns := range knownSystems {
		assert.True(t, systemNamespaces[ns], "%s should be system namespace", ns)
	}

	// Verify non-system
	nonSystem := []string{"default", "production", "staging", "test", "my-app"}
	for _, ns := range nonSystem {
		assert.False(t, systemNamespaces[ns], "%s should NOT be system namespace", ns)
	}
}

// ========== parseCapacityProvisioned 边界用例 ==========

func TestParseCapacityProvisioned_EdgeCases(t *testing.T) {
	// 仅vCPU部分
	cpu, mem := parseCapacityProvisioned("8vCPU")
	assert.InDelta(t, 8.0, cpu, 0.001)
	assert.InDelta(t, 0.0, mem, 0.01)

	// 仅GB部分
	cpu, mem = parseCapacityProvisioned("16GB")
	assert.InDelta(t, 0.0, cpu, 0.001)
	assert.InDelta(t, 16384.0, mem, 0.01)

	// 空字符串
	cpu, mem = parseCapacityProvisioned("")
	assert.InDelta(t, 0.0, cpu, 0.001)
	assert.InDelta(t, 0.0, mem, 0.01)

	// 小数vCPU
	cpu, mem = parseCapacityProvisioned("0.5vCPU 2GB")
	assert.InDelta(t, 0.5, cpu, 0.001)
	assert.InDelta(t, 2048.0, mem, 0.01)
}
