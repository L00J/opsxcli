# opsxcli Agent 内存管理系统

## 设计概述

Agent的内存管理采用分层设计，将记忆分为**短暂记忆**和**永久记忆**两层，总容量限制为500MB。

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    内存管理系统 (500MB)                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌────────────────────┐        ┌────────────────────────┐  │
│  │  短暂记忆 (100MB)   │        │   永久记忆 (400MB)      │  │
│  │                    │        │                        │  │
│  │  • 会话上下文       │        │  • 知识库              │  │
│  │  • 对话历史         │        │  • 用户偏好            │  │
│  │  • 工具使用统计     │        │  • 环境信息            │  │
│  │  • 当前任务状态     │        │  • 操作模式            │  │
│  │                    │        │                        │  │
│  │  生命周期: 会话     │        │  生命周期: 持久化      │  │
│  │  存储: 内存缓存     │        │  存储: SQLite          │  │
│  └────────────────────┘        └────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 一、短暂记忆（Short-term Memory）

### 容量分配
- **大小**: 100MB
- **存储位置**: 内存中的map缓存
- **生命周期**: 会话级别，会话结束后清空或压缩为摘要

### 存储内容

#### 1. 对话上下文 (ConversationContext)
```go
type ConversationContext struct {
    SessionID    string          // 会话ID
    Messages     []llm.Message   // 对话历史（最近50条）
    Summary      string          // 历史对话摘要
    ToolUsage    map[string]int  // 工具使用统计
    CreatedAt    time.Time       // 创建时间
    UpdatedAt    time.Time       // 更新时间
    MaxMessages  int             // 最大消息数(50)
}
```

#### 2. 消息压缩机制
当对话超过50条消息时，自动压缩：

1. **保留系统消息**: 所有system角色的消息
2. **保留最近消息**: 最近20条消息
3. **压缩历史消息**: 将旧消息生成摘要

压缩示例：
```
原始: 100条消息 (5MB)
↓
压缩后:
  - 系统消息: 3条
  - 历史摘要: "历史操作: ls, cat, grep, kubectl_get (共25次工具调用)"
  - 最近消息: 20条
总计: ~1MB
```

### 使用场景
- LLM上下文窗口管理
- 当前会话的工具调用历史
- 临时任务状态跟踪
- 最近的操作结果

### API示例
```go
// 创建内存管理器
memMgr := memory.NewMemoryManager(database)

// 添加消息
memMgr.AddMessage(sessionID, llm.Message{
    Role: "user",
    Content: "查看磁盘使用情况",
})

// 获取最近消息（用于LLM上下文）
recentMsgs := memMgr.GetRecentMessages(sessionID, 20)

// 记录工具使用
memMgr.RecordToolUsage(sessionID, "df")

// 清除会话
memMgr.ClearSession(sessionID)
```

## 二、永久记忆（Long-term Memory）

### 容量分配
- **大小**: 400MB
- **存储位置**: SQLite数据库
- **生命周期**: 持久化，除非主动清理

### 存储内容

#### 1. 知识库 (Knowledge)
存储常见问题和解决方案

```go
// 保存知识
memMgr.SaveKnowledge("troubleshooting", "high_cpu", `
{
  "problem": "CPU使用率过高",
  "steps": [
    "使用top查看进程",
    "找出CPU占用高的进程",
    "分析进程是否正常",
    "考虑重启或优化"
  ],
  "tools": ["top", "ps", "kill"]
}
`)

// 获取知识
solution, _ := memMgr.GetKnowledge("troubleshooting", "high_cpu")
```

知识库分类：
- `troubleshooting` - 故障排查
- `best_practice` - 最佳实践
- `command_pattern` - 命令模式
- `environment_info` - 环境信息

#### 2. 用户偏好 (Preference)
存储用户的习惯和偏好

```go
// 保存偏好
memMgr.SavePreference("safety_mode", "balanced")
memMgr.SavePreference("default_provider", "deepseek")
memMgr.SavePreference("favorite_tools", `["kubectl", "grep", "cat"]`)

// 获取偏好
mode, _ := memMgr.GetPreference("safety_mode")
```

偏好类型：
- `safety_mode` - 安全模式（strict/balanced/permissive）
- `default_provider` - 默认LLM提供商
- `favorite_tools` - 常用工具列表
- `notification_level` - 通知级别

#### 3. 环境信息 (Environment)
存储环境相关的配置和信息

```go
// 保存环境信息
memMgr.SaveEnvironmentInfo("k8s_clusters", `[
  {"name": "prod", "context": "prod-cluster"},
  {"name": "test", "context": "test-cluster"}
]`)

memMgr.SaveEnvironmentInfo("ssh_hosts", `[
  {"alias": "web01", "host": "192.168.1.10"},
  {"alias": "db01", "host": "192.168.1.20"}
]`)

// 获取环境信息
clusters, _ := memMgr.GetEnvironmentInfo("k8s_clusters")
```

环境信息类型：
- `k8s_clusters` - Kubernetes集群列表
- `ssh_hosts` - SSH主机列表
- `database_connections` - 数据库连接信息
- `api_endpoints` - API端点信息

### 数据库Schema

```sql
CREATE TABLE IF NOT EXISTS memories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK(type IN ('short_term', 'long_term')),
    session_id TEXT,
    category TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    metadata TEXT,
    access_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE(type, category, key),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);
```

## 三、容量管理

### 容量分配策略
```
总容量: 500MB
├── 短暂记忆: 100MB (20%)
│   └── 会话上下文 × N个会话
└── 永久记忆: 400MB (80%)
    ├── 知识库: ~200MB (40%)
    ├── 用户偏好: ~50MB (10%)
    └── 环境信息: ~150MB (30%)
```

### 自动清理机制

#### 1. 短暂记忆清理
- **触发条件**: 会话1小时无活动
- **清理策略**: 删除整个会话上下文
- **保留**: 可选择保存摘要到知识库

#### 2. 永久记忆清理
- **触发条件**: 总容量超过90% (450MB)
- **清理策略**:
  1. 清理最少访问的记录（access_count最低）
  2. 清理最旧的记录（updated_at最早）
  3. 每次清理100条记录

#### 3. 过期清理
- **触发条件**: 定期检查 (每小时)
- **清理策略**: 删除expires_at < NOW的记录

### 容量监控

```go
// 获取记忆统计
stats, _ := memMgr.GetMemoryStats()

fmt.Printf("短暂记忆: %.2f MB / 100 MB\n", stats["short_term_size_mb"])
fmt.Printf("永久记忆: %.2f MB / 400 MB\n", stats["long_term_size_mb"])
fmt.Printf("总使用: %.2f MB / 500 MB (%.1f%%)\n",
    stats["total_size_mb"],
    stats["usage_percent"])
```

输出示例：
```
短暂记忆: 45.23 MB / 100 MB
永久记忆: 234.56 MB / 400 MB
总使用: 279.79 MB / 500 MB (56.0%)
```

### 主动管理

```go
// 手动触发清理
memMgr.TrimMemoryIfNeeded()

// 清理过期记忆
memMgr.CleanExpiredMemories()

// 清理特定会话
memMgr.ClearSession(sessionID)
```

## 四、使用示例

### 场景1：问题排查学习

```go
// 用户遇到问题，Agent解决后保存为知识
solution := `{
  "problem": "Pod一直重启",
  "steps": [
    "kubectl get pods -n default",
    "kubectl logs <pod-name>",
    "kubectl describe pod <pod-name>",
    "检查镜像是否正常",
    "检查资源限制"
  ],
  "resolution": "镜像拉取失败，修复镜像仓库地址"
}`

memMgr.SaveKnowledge("k8s_troubleshooting", "pod_crashloop", solution)
```

### 场景2：用户偏好学习

```go
// Agent观察到用户总是使用strict模式
sessionCtx := memMgr.GetOrCreateContext(sessionID)

// 统计用户的安全模式选择
strictCount := 0
for _, msg := range sessionCtx.Messages {
    if strings.Contains(msg.Content, "safety=strict") {
        strictCount++
    }
}

// 如果用户多次使用strict，保存为偏好
if strictCount > 5 {
    memMgr.SavePreference("safety_mode", "strict")
}
```

### 场景3：环境信息复用

```go
// 首次获取k8s集群信息后保存
clusters := []string{"prod", "test", "dev"}
data, _ := json.Marshal(clusters)
memMgr.SaveEnvironmentInfo("k8s_clusters", string(data))

// 后续查询时直接从记忆获取
if cached, err := memMgr.GetEnvironmentInfo("k8s_clusters"); err == nil {
    var clusters []string
    json.Unmarshal([]byte(cached), &clusters)
    // 直接使用，无需重新查询
}
```

## 五、500MB容量分析

### 容量是否够用？

#### 典型使用场景估算：

1. **短暂记忆 (100MB)**
   - 单个会话上下文: ~200KB (50条消息)
   - 可支持会话数: 500个并发会话
   - **结论**: 绰绰有余（一般用户同时只有1-5个会话）

2. **永久记忆 (400MB)**
   - 知识条目: ~5KB/条 → 40,000条
   - 用户偏好: ~1KB/条 → 10,000条
   - 环境信息: ~10KB/条 → 10,000条
   - **结论**: 足够存储大量知识和配置

### 极端情况
- **重度用户**: 每天100次查询 × 365天 = 36,500条记录
- **单条记录**: ~10KB
- **总占用**: ~350MB
- **结论**: 一年的重度使用仍在限制内

### 扩展方案
如果未来容量不足，可以采取：
1. 增加总容量限制到1GB
2. 实现分级存储（热数据在内存，冷数据在磁盘）
3. 引入向量数据库进行语义搜索压缩

## 六、最佳实践

### 1. 会话管理
- 及时清理不用的会话
- 长时间会话定期压缩
- 重要会话保存摘要到知识库

### 2. 知识沉淀
- 成功的解决方案保存为知识
- 定期review和更新知识库
- 使用有意义的key命名

### 3. 偏好优化
- 观察用户行为自动调整
- 提供偏好重置功能
- 偏好变更时记录原因

### 4. 容量监控
- 定期检查容量使用
- 设置告警阈值（80%）
- 主动清理过期数据

## 总结

opsxcli Agent的内存管理系统设计合理，500MB容量对于大多数使用场景完全足够：

✅ **短暂记忆**: 100MB可支持500个并发会话
✅ **永久记忆**: 400MB可存储数万条知识和配置
✅ **自动清理**: 智能清理机制防止容量溢出
✅ **可扩展**: 未来可根据需求轻松扩容

这个设计既保证了性能，又提供了足够的存储空间用于Agent的学习和优化。
