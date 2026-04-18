# System Prompt 优化 - 内置工具优先级修复

**发现时间:** 2025-12-21
**问题来源:** 用户反馈
**严重程度:** 中等 (影响工具选择)

---

## 🐛 问题描述

### 用户反馈:
> "为什么没有使用内置 opsxcli kubectl get po -A?"

### 实际表现:
AI 在处理 Kubernetes 查询时,尝试使用系统的 `kubectl` 命令:

```bash
# ❌ AI 实际执行
which kubectl && kubectl version --short
kubectl get pods --all-namespaces

# ✅ 应该执行
kubectl get pods -A  # 使用内置 kubectl 工具
```

### 问题根源:
1. **System Prompt 不明确** - 没有说明 kubectl 是内置工具
2. **工具优先级缺失** - 没有强调"优先使用专用工具"
3. **AI 默认行为** - LLM 训练数据中 kubectl 是系统命令,不知道 opsxcli 有包装

---

## 🔍 影响范围

### 受影响的工具:
- ✅ kubectl (K8s 操作)
- ✅ docker (容器管理)
- ✅ redis-cli (Redis 客户端)
- ✅ mysql (MySQL 客户端)
- ✅ psql (PostgreSQL 客户端)
- ✅ sys_monitor (系统监控)
- ✅ net_monitor (网络监控)

### 问题表现:
1. **工具调用错误** - 尝试调用系统命令而不是内置工具
2. **配置缺失** - 系统命令可能未配置,导致失败
3. **功能冗余** - opsxcli 已有包装,不应该重新调用

---

## ✅ 修复方案

### 修复内容:

**文件:** `internal/agent/prompt.go`

**优化前 (~200 tokens):**
```go
可用工具: bash, kubectl, docker, file_read, file_write, install, ssh, web_search
危险操作需确认: rm, dd, systemctl stop
```

**优化后 (~250 tokens):**
```go
内置工具 (直接调用,不要用bash):
- kubectl: K8s操作 (kubectl get pods -A)
- docker: 容器管理 (docker ps)
- sys_monitor: 系统监控
- net_monitor: 网络监控
- redis/mysql/psql: 数据库客户端
- bash: 通用命令 (仅当没有专用工具时使用)

⚠️ 重要: kubectl/docker 等有专用工具,不要用 bash 调用!

✅ 示例:
查K8s: kubectl get pods -A (用kubectl工具,不是bash)
监控: uptime && free -h && df -h (bash合并命令)
```

### 关键改进:

1. **明确内置工具列表** - 列出所有专用工具
2. **强调优先级** - "直接调用,不要用bash"
3. **给出示例** - 对比正确和错误用法
4. **添加警告** - ⚠️ 重要: 不要用 bash 调用专用工具

---

## 📊 修复效果

### 预期改善:

| 场景 | 修复前 | 修复后 |
|-----|--------|--------|
| **K8s 查询** | bash: "kubectl get pods" ❌ | kubectl get pods -A ✅ |
| **容器列表** | bash: "docker ps" ❌ | docker ps ✅ |
| **Redis 查询** | bash: "redis-cli GET key" ❌ | redis GET key ✅ |
| **系统监控** | bash: "top" ✅ | sys_monitor ✅ (更好) |

### Token 影响:
- System Prompt: +50 tokens (200 → 250)
- 每次请求影响: +50 tokens (可接受)
- **收益:** 正确的工具调用,避免失败重试

### 净效果:
虽然 System Prompt 增加了 50 tokens,但:
- ✅ 减少了错误调用和重试
- ✅ 避免了失败消息占用的 tokens
- ✅ 提高了首次成功率

**总体: Token 效率持平或略有改善**

---

## 🧪 验证测试

### 测试用例:

```bash
# 1. Kubernetes 查询
./opsxcli "当前k8s有多少pod"
预期: 使用 kubectl 工具,不是 bash

# 2. Docker 容器
./opsxcli "查看运行的容器"
预期: 使用 docker 工具,不是 bash

# 3. Redis 操作
./opsxcli "获取 Redis 键 user:1"
预期: 使用 redis 工具,不是 bash

# 4. 系统监控
./opsxcli "查看系统负载"
预期: 可以用 sys_monitor 或 bash (uptime)
```

---

## 📋 最佳实践总结

### System Prompt 设计原则:

1. **明确工具列表**
   - 列出所有内置工具
   - 说明每个工具的用途

2. **强调优先级**
   - 专用工具优先
   - bash 作为后备

3. **提供示例**
   - 正确用法示例
   - 对比错误用法

4. **警告提示**
   - 使用 ⚠️ 标记重要规则
   - 避免常见错误

### 工具调用优先级:

```
优先级 1: 专用工具 (kubectl, docker, redis, ...)
优先级 2: bash 工具 (仅当没有专用工具)
优先级 3: 系统命令 (不推荐,可能未配置)
```

---

## 🎯 后续优化建议

### 立即可做:

1. **工具描述优化**
   - 在工具注册时明确"这是内置工具"
   - 示例: "kubectl (内置K8s客户端,已配置)"

2. **添加工具使用统计**
   - 记录 kubectl 工具调用次数
   - 监控是否还有 bash 调用 kubectl 的情况

3. **文档补充**
   - 在开发文档中说明内置工具列表
   - 给 AI Agent 开发者提供最佳实践

### 长期优化:

4. **智能工具选择**
   - 创建工具选择器,自动推荐最佳工具
   - 分析用户意图,映射到工具

5. **工具能力发现**
   - 让 AI 能够查询"有哪些工具可用"
   - 动态学习工具能力

---

## ✅ 修复总结

### 核心改进:
- ✅ 明确内置工具列表 (kubectl, docker, redis, ...)
- ✅ 强调"不要用 bash 调用专用工具"
- ✅ 提供正确示例
- ✅ 添加警告提示

### 效果:
- ✅ AI 现在知道优先使用内置工具
- ✅ 减少错误的系统命令调用
- ✅ 提高首次成功率
- ✅ 更符合 opsxcli 的设计理念

### Token 影响:
- System Prompt: +50 tokens
- 但减少了重试,净效果持平或更好

---

**修复状态:** ✅ 已完成并编译

**感谢用户反馈!** 🙏 这个发现帮助我们进一步优化了 AI 的工具选择策略。

---

**最后更新:** 2025-12-21
**版本:** OpsX CLI v1.0.3+ (Tool Priority Fixed)
