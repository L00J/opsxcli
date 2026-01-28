package tools

import (
	"context"
	"fmt"
	"strings"

	"opsxcli/plugins/websearch"
)

// WebSearchTool Web搜索工具
type WebSearchTool struct {
	name        string
	description string
	parameters  map[string]interface{}
}

// NewWebSearchTool 创建Web搜索工具
func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		name: "web_search",
		description: `Web搜索工具 - 仅在以下6种场景使用:
1. 实时信息: 软件最新版本、CVE漏洞、服务状态 (包含"最新"/"2025"/"CVE"等关键词)
2. 官方文档: 需要权威API规范或配置说明 (包含"官方"/"official"/"documentation")
3. 安全合规: 安全漏洞、合规要求 (不能有错误)
4. 资料爬虫: 数据采集、网页监控 (包含"爬取"/"crawl"/"采集")
5. 社区案例: GitHub Issues、实际踩坑经验 (包含"issue"/"案例"/"实践")
6. 多源对比: 技术选型、方案对比 (需要多方观点)

⚠️ 注意:
- 优先使用现有知识和本地工具(kubectl/docker/bash等)
- 概念解释、代码示例、配置生成等场景不需要搜索
- 仅在上述6种不可替代场景使用此工具
- 默认使用百度+Google,国内问题优先百度`,
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词或问题",
				},
				"engines": map[string]interface{}{
					"type":        "string",
					"description": "搜索引擎列表(逗号分隔,可选: baidu,google,bing,sogou,so360,bing_cn,duckduckgo,yahoo)。默认使用 baidu,google",
				},
			},
			"required": []string{"query"},
		},
	}
}

func (t *WebSearchTool) Name() string {
	return t.name
}

func (t *WebSearchTool) Description() string {
	return t.description
}

func (t *WebSearchTool) Parameters() map[string]interface{} {
	return t.parameters
}

func (t *WebSearchTool) RiskLevel() RiskLevel {
	return RiskSafe // Web搜索是安全操作
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	// 解析参数
	query := parseStringParam(args, "query")
	if query == "" {
		return &ToolResult{
			Success: false,
			Error:   "query 参数不能为空",
		}, nil
	}

	// 解析搜索引擎列表
	enginesStr := parseStringParam(args, "engines")
	var engines []string
	if enginesStr != "" {
		engines = strings.Split(enginesStr, ",")
		for i := range engines {
			engines[i] = strings.TrimSpace(engines[i])
		}
	}

	// 执行搜索 (不保存HTML)
	err := websearch.Search(query, engines, false)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("搜索失败: %v", err),
		}, nil
	}

	// 构建成功消息
	engineNames := getEngineNames(engines)
	if len(engineNames) == 0 {
		engineNames = []string{"百度", "Google"} // 默认
	}

	output := fmt.Sprintf("已在 %s 中搜索 \"%s\",请查看上方结果",
		strings.Join(engineNames, ", "), query)

	return &ToolResult{
		Success: true,
		Output:  output,
	}, nil
}

// getEngineNames 获取搜索引擎名称列表
func getEngineNames(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}

	names := make([]string, 0, len(keys))
	for _, key := range keys {
		if engine, ok := websearch.SearchEngines[key]; ok {
			names = append(names, engine.Name)
		} else {
			names = append(names, key)
		}
	}
	return names
}
