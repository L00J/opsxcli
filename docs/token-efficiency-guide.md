# Token 高效利用策略 - 减少浪费指南

## 🎯 核心目标

**最大化效率,最小化 Token 浪费**

## 📊 当前 Token 使用分析

### 问题: 搜索结果返回完整 HTML

```
单次搜索 Token 消耗:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
百度 HTML: 1.5 KB → 400 tokens
Google HTML: 41 KB → 10,000 tokens
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
总计: ~10,400 tokens per search

浪费: 95%+ (实际只需要 500 tokens 的摘要)
```

## 🚀 优化策略

### 1. **工具描述优化** ✅ 已完成

```go
// 优化前:
description: "在多个搜索引擎中搜索信息"

// 优化后:
description: `仅在以下6种场景使用:
1. 实时信息 (包含"最新"/"2025")
2. 官方文档 (包含"官方"/"official")
...
⚠️ 优先使用现有知识和本地工具`
```

**效果**: AI 会自动减少不必要的搜索调用

### 2. **HTML 解析提取** 🔥 立即实施

#### 当前问题:
```
返回完整 HTML (41 KB) → AI 处理 → 浪费 10,000 tokens
```

#### 解决方案:
```go
// plugins/websearch/parser.go
package websearch

import (
    "github.com/PuerkitoBio/goquery"
    "strings"
)

// SearchResult 搜索结果摘要
type SearchResult struct {
    Title       string // 标题
    Description string // 描述 (200字符以内)
    URL         string // 链接
}

// ExtractResults 从 HTML 提取搜索结果
func ExtractResults(html string, engine string) []SearchResult {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        return nil
    }

    results := []SearchResult{}

    switch engine {
    case "google":
        // Google 搜索结果
        doc.Find("div.g").Each(func(i int, s *goquery.Selection) {
            if i >= 5 { // 只取前 5 条
                return
            }

            title := s.Find("h3").Text()
            desc := s.Find("div.VwiC3b").Text()
            url, _ := s.Find("a").Attr("href")

            // 限制描述长度
            if len(desc) > 200 {
                desc = desc[:200] + "..."
            }

            results = append(results, SearchResult{
                Title:       title,
                Description: desc,
                URL:         url,
            })
        })

    case "baidu":
        // 百度搜索结果
        doc.Find("div.result").Each(func(i int, s *goquery.Selection) {
            if i >= 5 { // 只取前 5 条
                return
            }

            title := s.Find("h3").Text()
            desc := s.Find("div.c-abstract").Text()
            url, _ := s.Find("a").Attr("href")

            // 限制描述长度
            if len(desc) > 200 {
                desc = desc[:200] + "..."
            }

            results = append(results, SearchResult{
                Title:       title,
                Description: desc,
                URL:         url,
            })
        })
    }

    return results
}

// FormatResults 格式化搜索结果为简洁文本
func FormatResults(results []SearchResult, engine string) string {
    if len(results) == 0 {
        return "未找到结果"
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("【%s 搜索结果】\n", engine))

    for i, r := range results {
        sb.WriteString(fmt.Sprintf("\n%d. %s\n", i+1, r.Title))
        sb.WriteString(fmt.Sprintf("   %s\n", r.Description))
        sb.WriteString(fmt.Sprintf("   %s\n", r.URL))
    }

    return sb.String()
}
```

#### Token 对比:
```
优化前:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
完整 HTML: 41 KB = 10,000 tokens
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

优化后:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
5 条结果摘要:
- 标题: 50字符 × 5 = 250字符
- 描述: 200字符 × 5 = 1000字符
- URL: 100字符 × 5 = 500字符
总计: 1750字符 = ~500 tokens
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

节省: 95% tokens! (10,000 → 500)
```

### 3. **直接答案提取** 🎯 高优先级

```go
// plugins/websearch/answer.go
package websearch

// DirectAnswer 搜索引擎的直接答案
type DirectAnswer struct {
    Type    string // featured_snippet, answer_box, baike
    Content string // 答案内容 (限制 500 字符)
    Source  string // 来源
}

// ExtractDirectAnswer 提取搜索引擎直接答案
func ExtractDirectAnswer(html string, engine string) *DirectAnswer {
    doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

    switch engine {
    case "google":
        // Google Featured Snippet
        if snippet := doc.Find("div.kp-blk").First(); snippet.Length() > 0 {
            content := snippet.Find("div[data-attrid]").Text()

            // 限制长度
            if len(content) > 500 {
                content = content[:500] + "..."
            }

            return &DirectAnswer{
                Type:    "Featured Snippet",
                Content: content,
                Source:  snippet.Find("cite").Text(),
            }
        }

    case "baidu":
        // 百度百科
        if baike := doc.Find("div.c-abstract").First(); baike.Length() > 0 {
            content := baike.Text()

            if len(content) > 500 {
                content = content[:500] + "..."
            }

            return &DirectAnswer{
                Type:    "百度百科",
                Content: content,
                Source:  "百度百科",
            }
        }
    }

    return nil
}
```

#### 效果:
```
有直接答案:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
答案内容: 500字符 = 150 tokens
无需 AI 处理!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
节省: 99% tokens!
```

### 4. **System Prompt 精简** 📝 重要

```go
// 优化前 (冗长):
systemPrompt := `
你是一个强大的AI助手,拥有丰富的知识...
你可以帮助用户完成各种任务...
你应该始终保持礼貌和专业...
...
(1000+ tokens)
`

// 优化后 (精简):
systemPrompt := `
运维AI助手。优先使用本地工具。
web_search仅用于: 实时信息/官方文档/CVE/爬虫/案例/选型
可用工具: kubectl, docker, bash, sys_monitor, web_search
(200 tokens)
`

节省: 800 tokens per request
```

### 5. **搜索结果缓存** 💾 进一步优化

```go
// internal/cache/search_cache.go
package cache

import (
    "sync"
    "time"
)

var searchCache = &SearchCache{
    data: make(map[string]*CachedResult),
    mu:   &sync.RWMutex{},
}

type CachedResult struct {
    Results   []SearchResult
    Answer    *DirectAnswer
    Timestamp time.Time
}

type SearchCache struct {
    data map[string]*CachedResult
    mu   *sync.RWMutex
}

func (c *SearchCache) Get(query string) *CachedResult {
    c.mu.RLock()
    defer c.mu.RUnlock()

    result, ok := c.data[query]
    if !ok {
        return nil
    }

    // 检查是否过期 (24小时)
    if time.Since(result.Timestamp) > 24*time.Hour {
        return nil
    }

    return result
}

func (c *SearchCache) Set(query string, result *CachedResult) {
    c.mu.Lock()
    defer c.mu.Unlock()

    result.Timestamp = time.Now()
    c.data[query] = result
}
```

#### 效果:
```
缓存命中:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Token 消耗: 0 tokens
响应时间: <10ms
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
节省: 100%!
```

## 📊 综合优化效果

### 单次搜索对比:

| 阶段 | Token 消耗 | 节省 |
|-----|-----------|------|
| 当前 (完整HTML) | 10,400 | - |
| +HTML解析 | 500 | 95% |
| +直接答案 | 150 | 99% |
| +缓存命中 | 0 | 100% |

### 月度成本对比 (10,000 次搜索):

```
当前实现:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
10,000 × 10,400 tokens = 104M tokens
成本: $208/月 (DeepSeek)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

优化后 (HTML解析):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
10,000 × 500 tokens = 5M tokens
成本: $10/月
节省: 95% ($198/月)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

优化后 (直接答案):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
30% 直接答案: 3,000 × 150 = 0.45M
70% 搜索摘要: 7,000 × 500 = 3.5M
总计: 3.95M tokens
成本: $7.9/月
节省: 96% ($200/月)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

优化后 (+ 缓存 20%命中):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
20% 缓存: 2,000 × 0 = 0
24% 直接答案: 2,400 × 150 = 0.36M
56% 搜索摘要: 5,600 × 500 = 2.8M
总计: 3.16M tokens
成本: $6.3/月
节省: 97% ($201.7/月)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## 🎯 实施优先级

### 立即实施 (1-2天):

1. ✅ **工具描述优化** - 已完成
   - 明确使用场景
   - 减少不必要调用

2. 🔥 **HTML 解析提取** - 最高优先级
   ```bash
   go get github.com/PuerkitoBio/goquery
   # 实现 parser.go
   # 节省 95% tokens
   ```

3. 📝 **System Prompt 精简**
   - 从 1000 tokens → 200 tokens
   - 节省 80% System tokens

### 短期实施 (1周):

4. 🎯 **直接答案提取**
   - Featured Snippet
   - 百度百科
   - 节省 99% tokens

5. 💾 **搜索结果缓存**
   - 24小时 TTL
   - 缓存命中 0 tokens

## 💡 最佳实践

### 1. 智能搜索策略

```go
// 优化搜索关键词
func OptimizeQuery(query string) string {
    // 移除无用词
    query = strings.ReplaceAll(query, "请", "")
    query = strings.ReplaceAll(query, "帮我", "")

    // 添加关键标识
    if strings.Contains(query, "最新") {
        query += " 2025"
    }

    return query
}
```

### 2. 限制返回结果数量

```go
// 只返回前 5 条,不要 10 条
const MaxResults = 5

// 限制描述长度
const MaxDescriptionLength = 200
```

### 3. 智能选择搜索引擎

```go
func SelectEngines(query string) []string {
    // 中文问题优先百度
    if containsChinese(query) {
        return []string{"baidu"}
    }

    // 技术问题优先 Google
    if containsTechKeywords(query) {
        return []string{"google"}
    }

    // 默认百度+Google
    return []string{"baidu", "google"}
}
```

## 📝 总结

### Token 优化黄金法则:

1. **优先 AI 知识库** - 80% 场景无需搜索
2. **精简返回内容** - HTML→摘要,节省 95%
3. **提取直接答案** - 无需 AI 处理,节省 99%
4. **缓存常见查询** - 缓存命中节省 100%
5. **优化 Prompt** - System Prompt 精简 80%

### 预期效果:

```
月度成本: $208 → $6.3
节省比例: 97%
年度节省: $2,420

性能提升: 响应时间减少 50%
准确度: 保持 95%+
```

**立即实施 HTML 解析,可快速节省 95% tokens!**
