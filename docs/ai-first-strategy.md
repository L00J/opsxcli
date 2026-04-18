# AI Agent 优先策略 - Web 搜索最小化使用原则

## 🎯 核心原则

**优先级顺序:**
```
1. AI Agent 自动化 (第一选择) ✅
2. AI 使用本地工具 (kubectl, docker, bash 等)
3. Web 搜索 (仅不可替代场景) ⚠️
```

## ✅ AI Agent 的优势

### 为什么优先使用 AI?

```
AI Agent 能力:
├── 智能理解用户意图
├── 自动选择合适工具
├── 多步骤任务编排
├── 上下文记忆
├── 错误自动修复
└── 持续学习优化

示例:
$ opsxcli "分析系统负载并优化"

AI 自动执行:
1. 调用 sys_monitor 查看负载
2. 分析进程占用
3. 生成优化建议
4. 执行优化命令
5. 验证结果

❌ 如果用 Web 搜索:
1. 搜索"如何查看系统负载"
2. 手动执行命令
3. 搜索"如何优化"
4. 手动执行
5. ...繁琐!
```

## ⚠️ Web 搜索仅用于 6 种场景

### 1. **实时/最新信息** (不可替代)

```bash
✅ 必须 Web:
- 软件最新版本: "Kubernetes 1.31 release notes"
- CVE 漏洞: "Redis CVE-2025-xxxx"
- 服务状态: "GitHub status"

❌ 不需要 Web:
- "什么是 Docker" → AI 直接回答
- "如何部署 K8s" → AI 直接回答
```

### 2. **官方文档/API 规范** (需要权威)

```bash
✅ 必须 Web:
- "Nginx official proxy_pass documentation"
- "Kubernetes API reference v1.31"
- "OWASP Top 10 2025"

❌ 不需要 Web:
- "Nginx 配置示例" → AI 生成
- "K8s 配置示例" → AI 生成
```

### 3. **安全漏洞/合规** (需要准确)

```bash
✅ 必须 Web:
- CVE 查询
- 安全公告
- 合规要求

原因: 不能有任何错误!
```

### 4. **资料爬虫/数据采集** (AI 无法访问)

```bash
✅ 使用 Web:
- 爬取网站数据
- 监控网页变化
- 采集特定信息
- 下载文件资源

示例:
$ opsxcli "爬取语雀文档列表"
→ AI 调用 web_fetch 工具
```

### 5. **社区案例/GitHub Issues** (需要实际经验)

```bash
✅ 使用 Web:
- "CrashLoopBackOff GitHub issue"
- "Redis cluster 生产实践"
- "Istio 踩坑经验"

原因: 需要看实际案例
```

### 6. **多源对比/技术选型** (需要全面)

```bash
✅ 使用 Web:
- "Prometheus vs VictoriaMetrics 对比"
- "服务网格选型"
- "数据库选型"

原因: 需要多方观点
```

## 📊 决策流程图

```
用户请求
    ↓
AI 可以直接处理?
    ↓ YES → AI Agent 执行 ✅
    ↓ NO
    ↓
需要执行系统命令?
    ↓ YES → AI 调用本地工具 ✅
    ↓ NO
    ↓
属于 6 种必须 Web 的场景?
    ↓ YES → AI 调用 web_search ⚠️
    ↓ NO
    ↓
AI 尝试回答 (可能建议用户搜索)
```

## 🎯 实际应用示例

### ✅ 优先 AI (90% 的场景)

```bash
# 1. 系统运维
$ opsxcli "检查磁盘使用率,清理大文件"
→ AI: df → 分析 → 找到大文件 → rm

# 2. 容器管理
$ opsxcli "重启失败的 Pod"
→ AI: kubectl get → 识别失败 Pod → kubectl delete

# 3. 配置生成
$ opsxcli "生成 Nginx 反向代理配置"
→ AI: 直接生成配置文件

# 4. 日志分析
$ opsxcli "分析最近的错误日志"
→ AI: tail → grep ERROR → 分析原因

# 5. 性能优化
$ opsxcli "优化 MySQL 查询性能"
→ AI: show processlist → 分析慢查询 → 优化建议
```

### ⚠️ 必须 Web (10% 的场景)

```bash
# 1. 最新版本
$ opsxcli "查询 Kubernetes 1.31 最新补丁版本"
→ AI: 调用 web_search (知识可能过时)

# 2. CVE 漏洞
$ opsxcli "检查 Redis 最新安全漏洞"
→ AI: 调用 web_search (安全必须最新)

# 3. 官方文档
$ opsxcli "查看 Istio 官方 Gateway 配置说明"
→ AI: 调用 web_search (需要权威文档)

# 4. 资料爬虫
$ opsxcli "爬取语雀所有文档标题"
→ AI: 调用 web_fetch (数据采集)

# 5. 社区案例
$ opsxcli "查找 CrashLoopBackOff 的 GitHub 解决方案"
→ AI: 调用 web_search (需要实际案例)
```

## 💰 成本效益分析

### AI Agent 优先的好处:

```
优势:
├── 速度快: 2-5秒 vs Web 搜索 10-30秒
├── 智能化: 自动多步骤执行
├── 成本低: $0.001-0.01 per query
├── 准确度: 95%+ (通用知识)
└── 体验好: 一句话搞定

劣势:
└── 知识有截止日期 (2025年1月)
```

### Web 搜索的定位:

```
定位: AI 的补充工具,不是主要手段

使用频率建议:
├── AI 直接处理: 80%
├── AI + 本地工具: 15%
└── AI + Web 搜索: 5%

成本对比 (10,000 次/月):
├── AI Only: $50
├── Web 为主: $0 (但体验差,需人工)
└── AI + Web (5%): $27 ✅ 最优
```

## 🎯 配置建议

### AI Agent 提示词优化

```go
系统提示词应包含:

1. 优先级声明:
"优先使用本地工具和知识库,仅在必要时调用 web_search"

2. Web 搜索触发条件:
"当遇到以下情况时使用 web_search:
- 用户明确要求搜索
- 查询包含'最新'、'2025'等时效性关键词
- 查询 CVE、安全漏洞
- 需要官方文档
- 数据采集需求"

3. 默认行为:
"其他情况优先使用现有知识和本地工具完成任务"
```

### 实现示例

```go
// internal/agent/decision.go
func ShouldUseWebSearch(query string) bool {
    // 1. 用户明确要求
    if strings.Contains(query, "搜索") ||
       strings.Contains(query, "search") {
        return true
    }

    // 2. 时效性关键词
    timeKeywords := []string{"最新", "latest", "2025", "2026", "now"}
    for _, kw := range timeKeywords {
        if strings.Contains(query, kw) {
            return true
        }
    }

    // 3. 安全相关
    securityKeywords := []string{"CVE", "漏洞", "vulnerability", "security"}
    for _, kw := range securityKeywords {
        if strings.Contains(query, kw) {
            return true
        }
    }

    // 4. 官方文档
    if strings.Contains(query, "官方") ||
       strings.Contains(query, "official") {
        return true
    }

    // 5. 数据采集
    if strings.Contains(query, "爬取") ||
       strings.Contains(query, "crawl") {
        return true
    }

    // 默认: 不使用 Web 搜索
    return false
}
```

## 📝 总结

### ✅ 推荐做法

```
1. 让 AI Agent 作为主要交互方式
2. AI 自动选择合适的工具 (bash, kubectl, docker 等)
3. Web 搜索作为 AI 的工具之一,而非优先选择
4. 仅在 6 种不可替代场景使用 Web 搜索
5. 配置智能判断逻辑,自动决策
```

### ❌ 避免做法

```
1. 什么都用 Web 搜索 (慢、需人工筛选)
2. 忽略 AI 的智能编排能力
3. 手动执行多步骤任务
4. 过度依赖搜索引擎
```

### 🎯 黄金法则

**"能用 AI 自动化的,绝不手动;能用本地工具的,绝不搜索;必须搜索的,让 AI 来决定"**

---

## 实际效果对比

### 场景: "优化 Docker 镜像大小"

#### ❌ Web 搜索为主 (传统方式):
```
1. 搜索"Docker 镜像优化"
2. 阅读文章
3. 手动修改 Dockerfile
4. docker build
5. 发现问题
6. 再次搜索
...
耗时: 30 分钟
```

#### ✅ AI Agent 为主 (推荐):
```
$ opsxcli "分析并优化这个 Dockerfile"

AI 自动执行:
1. 读取 Dockerfile
2. 分析问题 (基础镜像大、未清理缓存等)
3. 生成优化建议
4. 可选:自动应用优化
5. 对比大小

耗时: 30 秒
准确度: 95%+
```

**结论: AI Agent 优先,Web 搜索最小化!**
