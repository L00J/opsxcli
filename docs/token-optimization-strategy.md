# 大模型 Token 优化策略

## 🎯 核心理念

**通过 Web 搜索引擎获取答案,减少大模型的 Token 消耗**

## 💡 三大优化策略

### 1. **Rule 策略** - 系统提示词优化

当前实现在 `internal/agent/` 中,通过优化 System Prompt 减少 token:

```
❌ 冗长的提示词 (浪费 token):
你是一个强大的 AI 助手,拥有丰富的知识和经验...
你可以帮助用户完成各种任务...
你应该始终保持礼貌和专业...
(1000+ tokens)

✅ 精简的提示词 (节省 token):
运维 AI 助手,处理系统运维、容器编排、网络管理等。
可用工具: web_search, bash, kubectl...
(200 tokens)

节省: 800 tokens per request
```

### 2. **长内容压缩** - 搜索结果智能提取

当前问题: Web 搜索返回完整 HTML (40KB+)

```
当前实现:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
搜索引擎 → 完整 HTML → AI (浪费)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Google: 41.2 KB HTML
      ↓
AI 处理完整 HTML (10,000+ tokens)
      ↓
提取关键信息 (实际只需 500 tokens)

浪费: 9,500+ tokens per search
```

**优化方案:**

```go
// plugins/websearch/parser.go
package websearch

import (
    "github.com/PuerkitoBio/goquery"
    "strings"
)

// ExtractSearchResults 从 HTML 提取搜索结果
func ExtractSearchResults(html string, engine string) []SearchSnippet {
    doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

    snippets := []SearchSnippet{}

    switch engine {
    case "google":
        // Google 搜索结果在 <div class="g">
        doc.Find("div.g").Each(func(i int, s *goquery.Selection) {
            title := s.Find("h3").Text()
            description := s.Find("div.VwiC3b").Text()
            url, _ := s.Find("a").Attr("href")

            snippets = append(snippets, SearchSnippet{
                Title:       title,
                Description: description,
                URL:         url,
            })
        })

    case "baidu":
        // 百度搜索结果
        doc.Find("div.result").Each(func(i int, s *goquery.Selection) {
            title := s.Find("h3").Text()
            description := s.Find("div.c-abstract").Text()
            url, _ := s.Find("a").Attr("href")

            snippets = append(snippets, SearchSnippet{
                Title:       title,
                Description: description,
                URL:         url,
            })
        })
    }

    return snippets
}

type SearchSnippet struct {
    Title       string
    Description string
    URL         string
}
```

**效果对比:**

```
优化前:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
完整 HTML: 41.2 KB = ~10,000 tokens
AI 处理时间: 高
Token 成本: 高
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

优化后:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
提取结果: 10 条 × 200 字符 = ~500 tokens
AI 处理时间: 低
Token 成本: 低
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

节省: 95% tokens! (10,000 → 500)
```

### 3. **Web 搜索引擎答案提取** - 直接获取答案 ⭐

**核心创新: 搜索引擎已经提供了答案!**

#### 现代搜索引擎的答案功能:

| 搜索引擎 | 答案功能 | 示例 |
|---------|---------|------|
| Google | Featured Snippet | "What is Docker?" → 直接答案框 |
| Bing | Answer Box | "天气" → 天气卡片 |
| 百度 | 百度知道/百科 | "什么是 K8s" → 百科摘要 |
| DuckDuckGo | Instant Answer | "calculate 2+2" → 直接显示 4 |

#### 实现答案提取:

```go
// plugins/websearch/answer_extractor.go
package websearch

import (
    "github.com/PuerkitoBio/goquery"
    "strings"
)

// ExtractDirectAnswer 提取搜索引擎的直接答案
func ExtractDirectAnswer(html string, engine string) *DirectAnswer {
    doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

    switch engine {
    case "google":
        // Google Featured Snippet
        if featured := doc.Find("div.kp-blk").First(); featured.Length() > 0 {
            return &DirectAnswer{
                Type:    "featured_snippet",
                Content: featured.Find("div[data-attrid]").Text(),
                Source:  featured.Find("cite").Text(),
            }
        }

        // Google Answer Box
        if answer := doc.Find("div.Z0LcW").First(); answer.Length() > 0 {
            return &DirectAnswer{
                Type:    "answer_box",
                Content: answer.Text(),
            }
        }

    case "baidu":
        // 百度百科
        if baike := doc.Find("div.c-abstract").First(); baike.Length() > 0 {
            return &DirectAnswer{
                Type:    "baike",
                Content: baike.Text(),
                Source:  "百度百科",
            }
        }

    case "bing":
        // Bing Answer
        if answer := doc.Find("div.b_ans").First(); answer.Length() > 0 {
            return &DirectAnswer{
                Type:    "answer",
                Content: answer.Find("div.b_tpcn").Text(),
            }
        }
    }

    return nil
}

type DirectAnswer struct {
    Type    string  // featured_snippet, answer_box, baike, etc.
    Content string  // 答案内容
    Source  string  // 来源
}
```

#### 智能搜索策略:

```go
// internal/tools/websearch_tool.go

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
    query := args["query"].(string)

    // 1. 执行搜索
    results := websearch.Search(query, engines, false)

    // 2. 尝试提取直接答案
    for _, result := range results {
        if answer := websearch.ExtractDirectAnswer(result.HTML, result.Engine); answer != nil {
            // ✅ 找到直接答案,无需 AI 处理!
            return &ToolResult{
                Success: true,
                Output:  formatDirectAnswer(answer),
            }, nil
        }
    }

    // 3. 没有直接答案,提取搜索结果摘要
    snippets := websearch.ExtractSearchResults(results)

    return &ToolResult{
        Success: true,
        Output:  formatSnippets(snippets),
    }, nil
}

func formatDirectAnswer(answer *DirectAnswer) string {
    return fmt.Sprintf(`
🎯 直接答案 (来自 %s):
%s

来源: %s
`, answer.Type, answer.Content, answer.Source)
}
```

## 📊 Token 节省对比

### 场景 1: "什么是 Docker?"

```
❌ 传统方式 (调用大模型):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
用户问题: 50 tokens
系统提示: 500 tokens
AI 回答: 300 tokens
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
总计: 850 tokens
成本: $0.0017 (DeepSeek)

✅ Web 搜索 + 答案提取:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
搜索引擎: 0 tokens (免费)
提取答案: 0 tokens (本地处理)
返回答案: 200 字符
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
总计: 0 tokens
成本: $0

节省: 100% tokens! 🎉
```

### 场景 2: "Docker 容器优化最佳实践"

```
❌ 当前实现 (完整 HTML):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
搜索结果 HTML: 40 KB × 2 = 80 KB = 20,000 tokens
AI 处理: 2,000 tokens
系统提示: 500 tokens
AI 回答: 1,000 tokens
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
总计: 23,500 tokens
成本: $0.047

✅ 优化后 (提取摘要):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
搜索结果摘要: 20 条 × 200 字符 = 1,000 tokens
AI 处理: 500 tokens
系统提示: 200 tokens (精简)
AI 回答: 1,000 tokens
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
总计: 2,700 tokens
成本: $0.0054

节省: 88.5% tokens! (23,500 → 2,700)
```

## 🚀 实现计划

### 阶段 1: 搜索结果提取 (立即实施)

```bash
# 1. 添加 HTML 解析库
go get github.com/PuerkitoBio/goquery

# 2. 创建结果提取器
# plugins/websearch/parser.go

# 3. 修改搜索工具
# internal/tools/websearch_tool.go
```

### 阶段 2: 直接答案提取 (核心优化)

```bash
# 1. 实现答案提取器
# plugins/websearch/answer_extractor.go

# 2. 支持多种答案格式:
- Featured Snippet (Google)
- Answer Box (Bing)
- 百度百科/知道
- 计算器结果
- 天气信息
- 货币转换

# 3. 智能降级:
有直接答案 → 直接返回 (0 tokens)
有搜索摘要 → 提取摘要 (1,000 tokens)
无结果 → 调用 AI (23,500 tokens)
```

### 阶段 3: 缓存机制 (进一步优化)

```go
// internal/cache/search_cache.go
package cache

import (
    "time"
    "sync"
)

var searchCache = &Cache{
    data: make(map[string]*CachedResult),
    ttl:  24 * time.Hour,
}

type CachedResult struct {
    Answer    *DirectAnswer
    Snippets  []SearchSnippet
    Timestamp time.Time
}

func GetCached(query string) *CachedResult {
    // 如果缓存命中,直接返回,0 tokens!
}
```

## 📈 预期收益

### Token 节省:

| 场景 | 当前 | 优化后 | 节省 |
|-----|------|-------|------|
| 简单问答 | 850 tokens | 0 tokens | 100% |
| 搜索类问题 | 23,500 tokens | 2,700 tokens | 88.5% |
| 缓存命中 | 23,500 tokens | 0 tokens | 100% |

### 成本节省 (DeepSeek 定价):

```
月调用量: 10,000 次

❌ 优化前:
10,000 × 23,500 tokens = 235M tokens
成本: $470/月

✅ 优化后:
- 简单问答 (30%): 3,000 × 0 = 0 tokens
- 搜索问题 (50%): 5,000 × 2,700 = 13.5M tokens
- 缓存命中 (20%): 2,000 × 0 = 0 tokens
总计: 13.5M tokens
成本: $27/月

节省: $443/月 (94.3%)
```

## 🎯 总结

通过三大策略的组合:

1. **Rule 策略**: 精简提示词 → 节省 60%
2. **内容压缩**: 提取摘要 → 节省 95%
3. **答案提取**: 直接使用搜索引擎答案 → 节省 100%

**综合节省: 90%+ tokens**

搜索引擎已经为我们做了大量工作,我们只需要聪明地提取和使用这些答案!🎉

---

## 🤔 什么时候必须使用 Web 搜索?

虽然 AI 大模型很强大,但在以下场景**必须使用 Web 搜索**:

### 1. **实时/最新信息** 🔥 最关键

| 场景 | AI 局限 | Web 搜索优势 |
|-----|---------|-------------|
| 软件最新版本 | 知识截止(2025年1月) | 实时更新 |
| CVE 漏洞 | 无法获取最新 | 安全公告 |
| 服务状态 | 静态知识 | 实时监控 |
| 技术新闻 | 过时信息 | 当天新闻 |

**示例:**
```bash
❌ AI: "Kubernetes 最新版本?" → "1.29 (2025年1月)"
✅ Web: "Kubernetes latest version 2025" → "1.31.2 (Dec 2025)"
```

### 2. **准确性验证** - 防止幻觉

AI 可能产生不准确信息,特定场景需要验证:

```bash
高风险操作(必须 Web 验证):
├── 生产环境配置
│   ├── Nginx 配置参数
│   ├── MySQL 性能调优
│   └── K8s 资源限制
│
├── 安全相关
│   ├── SSL 证书配置
│   ├── 防火墙规则
│   └── 加密算法选择
│
└── API 规范
    ├── 函数签名
    ├── 参数类型
    └── 返回值格式
```

### 3. **官方文档/权威来源**

```bash
需要官方文档的场景:
├── API 文档: "Kubernetes API reference"
├── 配置说明: "Nginx official configuration"
├── 安全建议: "OWASP Top 10 2025"
└── 最佳实践: "Docker official best practices"
```

### 4. **多源对比/社区讨论**

```bash
需要多源信息:
├── 技术选型: 查看多个对比文章
├── 踩坑经验: GitHub Issues, Stack Overflow
├── 用户评价: Reddit, Hacker News
└── 实际案例: 技术博客, 生产实践
```

### 5. **小众/专有知识**

```bash
AI 覆盖不足的领域:
├── 企业内部工具文档
├── 小众开源项目
├── 特定行业技术
└── 新兴技术(发布 < 6个月)
```

### 6. **需要引用来源**

```bash
需要可追溯性:
├── 技术报告
├── 架构设计文档
├── 安全审计
└── 合规检查
```

## 🎯 推荐策略

### 智能混合使用:

```go
// 伪代码: 智能选择
func AnswerQuestion(question string) string {
    // 1. 判断是否需要实时信息
    if IsRealtimeQuery(question) {
        // 关键词: "最新", "2025", "现在", "当前"
        return WebSearch(question)
    }

    // 2. 判断是否高风险操作
    if IsHighRisk(question) {
        // 关键词: "生产环境", "配置", "删除", "安全"
        aiAnswer := AskAI(question)
        webVerify := WebSearch(question)
        return Combine(aiAnswer, webVerify)
    }

    // 3. 判断是否需要官方文档
    if NeedsOfficialDoc(question) {
        // 关键词: "API", "官方文档", "参数", "规范"
        return WebSearch(question + " official documentation")
    }

    // 4. 其他情况,直接问 AI
    return AskAI(question)
}
```

### 实际应用示例:

```bash
# 场景 1: 概念理解 → AI 优先
$ opsxcli "什么是 Docker 容器?"
✅ AI 直接回答 (知识库足够)

# 场景 2: 最新版本 → Web 搜索
$ opsxcli "Docker 最新版本和新特性"
✅ 自动触发 web_search (检测到"最新")

# 场景 3: 配置验证 → AI + Web
$ opsxcli "Nginx 反向代理配置"
✅ AI 生成配置 → Web 验证官方文档

# 场景 4: 故障诊断 → 混合
$ opsxcli "Pod CrashLoopBackOff 如何解决"
✅ AI 分析 → Web 搜索类似案例 (GitHub Issues)
```

## 📊 成本效益分析

| 场景 | AI | Web | 推荐 | 原因 |
|-----|----|----|------|------|
| 概念理解 | $0.001 | $0 | AI | 快速准确 |
| 最新版本 | $0.001(可能错) | $0 | Web | 必须准确 |
| 代码示例 | $0.005 | - | AI | AI 擅长 |
| 官方文档 | $0.001 | $0 | Web | 需要权威 |
| 故障诊断 | $0.01 | $0 | 混合 | 综合信息 |
| 技术选型 | $0.02 | $0 | 混合 | 多源对比 |

**结论**:
- 通用知识 → AI (快速)
- 实时/权威 → Web (准确)
- 复杂问题 → 混合 (全面)

## 💡 最佳实践

### 1. AI 优先策略
```bash
# 快速查询,通用知识
opsxcli "Docker 和 Podman 的区别"
```

### 2. Web 优先策略
```bash
# 实时信息,官方文档
opsxcli websearch "Kubernetes 1.31 release notes" --engines google
```

### 3. 混合策略 (推荐)
```bash
# AI 自动判断是否需要搜索
opsxcli "分析最新的 Kubernetes CVE 漏洞"
# → AI 检测到"最新" → 自动触发 web_search → 综合回答
```

## 🎯 总结

**传统 Web 搜索不可替代的场景:**

1. ✅ **实时信息**: 版本、CVE、服务状态
2. ✅ **准确性验证**: 生产配置、安全操作
3. ✅ **官方文档**: API 规范、配置说明
4. ✅ **多源对比**: 技术选型、社区讨论
5. ✅ **小众知识**: 企业工具、新兴技术
6. ✅ **可追溯性**: 需要引用来源

**AI 大模型的优势场景:**

1. ✅ **概念理解**: 快速学习基础知识
2. ✅ **代码生成**: 示例代码、脚本编写
3. ✅ **问题诊断**: 日志分析、错误排查
4. ✅ **最佳实践**: 通用经验总结
5. ✅ **方案设计**: 架构建议、技术选型

**推荐做法:**
- 让 AI 自动判断是否需要 Web 搜索
- 关键操作 Web 验证
- 成本敏感场景优先 Web
- 快速查询优先 AI
