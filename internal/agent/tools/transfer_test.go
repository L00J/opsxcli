package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===================== UnifiedTransferTool 动态风险评估测试 =====================

func TestUnifiedTransferTool_RiskLevelForArgs(t *testing.T) {
	tool := NewUnifiedTransferTool()

	tests := []struct {
		name string
		args map[string]interface{}
		want RiskLevel
	}{
		{
			name: "本地复制_低风险",
			args: map[string]interface{}{"source": "/tmp/a", "destination": "/tmp/b"},
			want: RiskLow,
		},
		{
			name: "远程下载_中风险",
			args: map[string]interface{}{
				"source":      "/tmp/remote_file",
				"destination": "/tmp/local_file",
				"host":        "server1",
				"direction":   "download",
			},
			want: RiskMedium,
		},
		{
			name: "远程上传_高风险",
			args: map[string]interface{}{
				"source":      "/tmp/local_file",
				"destination": "/tmp/remote_file",
				"host":        "server1",
				"direction":   "upload",
			},
			want: RiskHigh,
		},
		{
			name: "远程无方向参数_默认中风险",
			args: map[string]interface{}{
				"source":      "/tmp/file",
				"destination": "/tmp/file2",
				"host":        "server1",
			},
			want: RiskMedium,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tool.RiskLevelForArgs(tt.args)
			assert.Equal(t, tt.want, got, "RiskLevelForArgs() = %v, want %v", got, tt.want)
		})
	}
}

// 测试 UnifiedTransferTool 实现了 DynamicRiskTool 接口
func TestUnifiedTransferTool_ImplementsDynamicRiskTool(t *testing.T) {
	tool := NewUnifiedTransferTool()

	var _ DynamicRiskTool = tool
}
