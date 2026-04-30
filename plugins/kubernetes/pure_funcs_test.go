package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// resource.go 补充测试
// ---------------------------------------------------------------------------

// TestExtractDeploymentName_ThreePartsNoHash 测试三段名称无哈希后缀的情况
func TestExtractDeploymentName_ThreePartsNoHash(t *testing.T) {
	assert.Equal(t, "my-app-v2", extractDeploymentName("my-app-v2-abc"))
}

// TestParseMemoryValue_ShortUnits 测试短单位解析（M/G/K/T）
func TestParseMemoryValue_ShortUnits(t *testing.T) {
	assert.Equal(t, 1.0, parseMemoryValue("1M"))
	assert.Equal(t, 1000.0, parseMemoryValue("1G"))
	assert.Equal(t, 1.0/1000.0, parseMemoryValue("1K"))
	assert.Equal(t, 1000000.0, parseMemoryValue("1T"))
}
