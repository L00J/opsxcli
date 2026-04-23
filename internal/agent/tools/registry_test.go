package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===================== Registry 测试 =====================

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskSafe, "safe"},
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskCritical, "critical"},
		{RiskLevel(99), "unknown"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.String())
	}
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	assert.NotNil(t, r)
	assert.NotNil(t, r.tools)
	assert.Empty(t, r.List())
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	tool := NewLocalBashTool()

	r.Register(tool)

	// 正常获取
	got, err := r.Get("local_bash")
	assert.NoError(t, err)
	assert.Equal(t, "local_bash", got.Name())

	// 不存在的工具
	_, err = r.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "工具未找到")
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(NewLocalBashTool())
	r.Register(NewSSHExecuteTool())

	list := r.List()
	assert.Len(t, list, 2)
	names := map[string]bool{}
	for _, tool := range list {
		names[tool.Name()] = true
	}
	assert.True(t, names["local_bash"])
	assert.True(t, names["ssh_execute"])
}

func TestRegistry_ToLLMTools(t *testing.T) {
	r := NewRegistry()
	r.Register(NewLocalBashTool())

	llmTools := r.ToLLMTools()
	assert.Len(t, llmTools, 1)
	assert.Equal(t, "function", llmTools[0].Type)
	assert.Equal(t, "local_bash", llmTools[0].Function.Name)
	assert.NotEmpty(t, llmTools[0].Function.Description)
	assert.NotNil(t, llmTools[0].Function.Parameters)
}

func TestRegistry_RegisterDefaults(t *testing.T) {
	r := NewRegistry()
	r.RegisterDefaults()

	list := r.List()
	assert.Len(t, list, 8)

	expectedNames := map[string]bool{
		// v0.5.0 统一工具
		"execute": true, "transfer": true,
		// 旧工具（向后兼容）
		"local_bash": true, "ssh_execute": true, "scp_transfer": true,
		// 分析和文件工具
		"analyze_output": true, "file_read": true, "file_search": true,
	}
	for _, tool := range list {
		assert.True(t, expectedNames[tool.Name()], "unexpected tool: %s", tool.Name())
	}
}

func TestRegistry_Close(t *testing.T) {
	r := NewRegistry()
	r.Register(NewLocalBashTool())
	// Close should not panic even if tools don't implement Closer
	r.Close()
}

// ===================== parseStringParam 测试 =====================

func TestParseStringParam(t *testing.T) {
	assert.Equal(t, "hello", parseStringParam(map[string]interface{}{"key": "hello"}, "key"))
	assert.Equal(t, "", parseStringParam(map[string]interface{}{"key": 123}, "key"))
	assert.Equal(t, "", parseStringParam(map[string]interface{}{}, "key"))
	assert.Equal(t, "", parseStringParam(nil, "key"))
}

// ===================== parseNumberParam 测试 =====================

func TestParseNumberParam(t *testing.T) {
	// int
	val, ok := parseNumberParam(map[string]interface{}{"n": 42}, "n")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	// float64
	val, ok = parseNumberParam(map[string]interface{}{"n": float64(42.7)}, "n")
	assert.True(t, ok)
	assert.Equal(t, 42, val)

	// int64
	val, ok = parseNumberParam(map[string]interface{}{"n": int64(99)}, "n")
	assert.True(t, ok)
	assert.Equal(t, 99, val)

	// 不存在
	_, ok = parseNumberParam(map[string]interface{}{}, "n")
	assert.False(t, ok)

	// 类型不对
	_, ok = parseNumberParam(map[string]interface{}{"n": "notanumber"}, "n")
	assert.False(t, ok)

	// nil map
	_, ok = parseNumberParam(nil, "n")
	assert.False(t, ok)
}

// ===================== parseFloat64Param 测试 =====================

func TestParseFloat64Param(t *testing.T) {
	// float64
	val, ok := parseFloat64Param(map[string]interface{}{"f": 3.14}, "f")
	assert.True(t, ok)
	assert.InDelta(t, 3.14, val, 0.001)

	// int → float64
	val, ok = parseFloat64Param(map[string]interface{}{"f": 7}, "f")
	assert.True(t, ok)
	assert.InDelta(t, 7.0, val, 0.001)

	// int64 → float64
	val, ok = parseFloat64Param(map[string]interface{}{"f": int64(100)}, "f")
	assert.True(t, ok)
	assert.InDelta(t, 100.0, val, 0.001)

	// 不存在
	_, ok = parseFloat64Param(map[string]interface{}{}, "f")
	assert.False(t, ok)

	// nil
	_, ok = parseFloat64Param(nil, "f")
	assert.False(t, ok)
}

// ===================== parseBoolParam 测试 =====================

func TestParseBoolParam(t *testing.T) {
	assert.True(t, parseBoolParam(map[string]interface{}{"b": true}, "b"))
	assert.False(t, parseBoolParam(map[string]interface{}{"b": false}, "b"))
	assert.False(t, parseBoolParam(map[string]interface{}{"b": "true"}, "b"))
	assert.False(t, parseBoolParam(map[string]interface{}{}, "b"))
	assert.False(t, parseBoolParam(nil, "b"))
}

// ===================== Tool 接口方法测试 =====================

func TestAnalyzeOutputTool_Metadata(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	assert.Equal(t, "analyze_output", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotNil(t, tool.Parameters())
	assert.Equal(t, RiskSafe, tool.RiskLevel())
}

func TestFileReadTool_Metadata(t *testing.T) {
	tool := NewFileReadTool()
	assert.Equal(t, "file_read", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotNil(t, tool.Parameters())
	assert.Equal(t, RiskSafe, tool.RiskLevel())
}

func TestFileSearchTool_Metadata(t *testing.T) {
	tool := NewFileSearchTool()
	assert.Equal(t, "file_search", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotNil(t, tool.Parameters())
	assert.Equal(t, RiskSafe, tool.RiskLevel())
}

func TestSCPTransferTool_Metadata(t *testing.T) {
	tool := NewSCPTransferTool()
	assert.Equal(t, "scp_transfer", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotNil(t, tool.Parameters())
	assert.Equal(t, RiskMedium, tool.RiskLevel())
}

// ===================== analyzeCompare 测试 =====================

func TestAnalyzeCompare_MissingSecondText(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, summary := tool.analyzeCompare("some text", "")
	assert.Contains(t, output, "需要两段文本")
	assert.Contains(t, summary, "缺少第二段文本")
}

func TestAnalyzeCompare_IdenticalTexts(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, summary := tool.analyzeCompare("line1\nline2\nline3", "line1\nline2\nline3")
	assert.Contains(t, output, "两段文本完全相同")
	assert.Contains(t, output, "3 行")
	assert.Contains(t, summary, "共同行3")
}

func TestAnalyzeCompare_DifferentLineCounts(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, _ := tool.analyzeCompare("a\nb\nc", "x\ny")
	assert.Contains(t, output, "文本1 多 1 行")
}

func TestAnalyzeCompare_DifferentCharCounts(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, _ := tool.analyzeCompare("abcde", "ab")
	assert.Contains(t, output, "文本1 多 3 字符")
}

func TestAnalyzeCompare_Text2MoreChars(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, _ := tool.analyzeCompare("ab", "abcde")
	assert.Contains(t, output, "文本2 多 3 字符")
}

func TestAnalyzeCompare_CommonLines(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, _ := tool.analyzeCompare("hello\nworld\nfoo", "hello\nbar\nworld")
	assert.Contains(t, output, "共同非空行: 2")
}

func TestAnalyzeCompare_NoCommonLines(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, _ := tool.analyzeCompare("aaa\nbbb", "ccc\nddd")
	assert.Contains(t, output, "共同非空行: 0")
}

func TestAnalyzeCompare_SameLineCount(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, _ := tool.analyzeCompare("a\nb", "c\nd")
	assert.Contains(t, output, "行数差异: 相同")
}

// ===================== analyzeSummary 边界测试 =====================

func TestAnalyzeSummary_SingleLineNoNewline(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	output, _ := tool.analyzeSummary("hello world")
	assert.Contains(t, output, "1 行")
}

// ===================== Execute 测试（不依赖外部服务） =====================

func TestAnalyzeOutputTool_Execute_AnalyzeMode(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output":        "hello world\nthis is a test",
		"analysis_type": "summary",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "总 行 数: 2")
}

func TestAnalyzeOutputTool_Execute_ErrorDetect(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output":        "Error: something went wrong",
		"analysis_type": "error_detect",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestAnalyzeOutputTool_Execute_KeyExtract(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output":        "listening on 192.168.1.1:8080",
		"analysis_type": "key_extract",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestAnalyzeOutputTool_Execute_DefaultMode(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output": "some output",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestAnalyzeOutputTool_Execute_CompareMode(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output":        "text one",
		"analysis_type": "compare",
		"context":       "text two",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestAnalyzeOutputTool_Execute_CompareWithoutContext(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output":        "text one",
		"analysis_type": "compare",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	// context 为空时应提示缺少第二段文本
	assert.Contains(t, result.Output, "需要两段文本")
}

func TestAnalyzeOutputTool_Execute_EmptyOutput(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output": "",
	})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "无内容可分析")
}

func TestAnalyzeOutputTool_Execute_UnsupportedType(t *testing.T) {
	tool := NewAnalyzeOutputTool()
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"output":        "some text",
		"analysis_type": "unsupported",
	})
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}
