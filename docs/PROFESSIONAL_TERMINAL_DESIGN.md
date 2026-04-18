# OpsXCLI 专业交互终端设计

## 设计目标

打造一个专业的 AI 驱动交互终端，支持:
- 运维操作自动化
- 代码开发和重构
- 实时交互和上下文补充
- 高效的会话管理

## 核心功能

### 1. 专业交互界面

```
┌─────────────────────────────────────────────────────────────┐
│  OpsXCLI - AI DevOps Assistant                     Session: default │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  👤 User:                                                   │
│  查看 k8s 集群中 CPU 使用最高的 5 个 Pod                    │
│                                                              │
│  🤖 Assistant:                                              │
│  我会帮你查询 CPU 使用最高的 Pod...                         │
│                                                              │
│    → kubectl_top_pods                                       │
│    ✓ 已获取所有 Pod 的资源使用情况                          │
│                                                              │
│  结果:                                                       │
│  1. nginx-app (namespace: prod) - 850m CPU                  │
│  2. redis-cache (namespace: cache) - 720m CPU               │
│  ...                                                         │
│                                                              │
│  ✻ Thinking… (3s · step 2/50 · ↓ 3.2k tokens)              │
│                                                              │
├─────────────────────────────────────────────────────────────┤
│  ⏵ [type to add context, tab to autocomplete, ctrl+c to cancel] │
└─────────────────────────────────────────────────────────────┘
```

### 2. 核心特性

#### A. 实时交互
- **底部固定输入框**: 类似 Claude CLI，执行过程中可随时输入
- **流式输出**: LLM 响应实时显示，提升体验
- **进度指示**: 清晰的执行进度和 token 消耗统计
- **动画效果**: Spinner 动画显示思考状态

#### B. 智能代码操作
```python
# 新增代码工具
- file_read: 读取文件内容
- file_write: 写入文件
- file_edit: 智能编辑文件（查找替换）
- git_status: 查看 Git 状态
- git_diff: 查看代码变更
- git_commit: 提交代码
- code_search: 在代码库中搜索
- code_analyze: 分析代码结构
- test_run: 运行测试
```

#### C. 会话管理
- **多会话支持**: 不同项目/任务独立会话
- **历史保存**: 自动保存对话历史
- **上下文恢复**: 恢复之前的会话状态
- **会话导出**: 导出为 Markdown 格式

#### D. 高效操作
- **快捷键**:
  - `Ctrl+C`: 中断当前任务
  - `Ctrl+D`: 退出
  - `Ctrl+L`: 清屏
  - `↑/↓`: 历史命令
  - `Tab`: 自动补全
  - `Ctrl+R`: 搜索历史
- **命令别名**: 自定义常用命令
- **批量操作**: 支持批处理脚本

### 3. 技术架构

```
┌─────────────────────────────────────────────────────┐
│                     用户界面层                       │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ TUI Manager  │  │ Input Handler│  │  Renderer │ │
│  └──────────────┘  └──────────────┘  └───────────┘ │
├─────────────────────────────────────────────────────┤
│                     交互控制层                       │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │Session Mgr   │  │Context Mgr   │  │Stream Mgr │ │
│  └──────────────┘  └──────────────┘  └───────────┘ │
├─────────────────────────────────────────────────────┤
│                     Agent 核心层                     │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐ │
│  │ Agent Core   │  │Tool Registry │  │ LLM Client│ │
│  └──────────────┘  └──────────────┘  └───────────┘ │
├─────────────────────────────────────────────────────┤
│                     工具执行层                       │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────────┐ │
│  │K8s工具│ │系统工具│ │网络工具│ │文件工具│ │代码工具│ │
│  └──────┘ └──────┘ └──────┘ └──────┘ └──────────┘ │
└─────────────────────────────────────────────────────┘
```

### 4. 实现计划

#### Phase 1: 增强交互界面 (当前)
- [x] 优化迭代机制 (20 → 50 次)
- [x] 改进进度显示
- [x] 添加 token 统计
- [ ] 实现底部固定输入框
- [ ] 添加快捷键支持
- [ ] 实现流式输出

#### Phase 2: 代码开发工具
- [ ] 文件操作工具集
- [ ] Git 集成
- [ ] 代码搜索和分析
- [ ] 测试运行器

#### Phase 3: 会话管理
- [ ] 多会话支持
- [ ] 历史保存和恢复
- [ ] 上下文智能管理
- [ ] 会话导出功能

#### Phase 4: 效率提升
- [ ] 命令补全
- [ ] 智能建议
- [ ] 批处理支持
- [ ] 性能优化

## 使用场景示例

### 场景 1: 运维问题排查
```bash
$ opsxcli -i

⏵ 生产环境的 nginx Pod 一直重启，帮我排查原因

✻ Thinking…
  → kubectl_get_pods (namespace: prod, selector: app=nginx)
  → kubectl_describe_pod (pod: nginx-app-7d8f9)
  → kubectl_logs (pod: nginx-app-7d8f9, tail: 100)

🤖 发现问题：内存 OOM，建议调整 memory limit

⏵ 帮我生成修复的 YAML 配置

✻ Thinking…
  → file_read (path: k8s/nginx-deployment.yaml)
  → file_edit (增加 memory limit 到 512Mi)

🤖 已生成修复配置，是否应用？[y/N]
```

### 场景 2: 代码开发
```bash
$ opsxcli -i

⏵ 重构 internal/agent/agent.go，抽取进度显示逻辑到单独的文件

✻ Thinking…
  → file_read (internal/agent/agent.go)
  → code_analyze (找出进度显示相关代码)
  → file_write (internal/agent/progress.go)
  → file_edit (更新 agent.go 使用新模块)

🤖 已完成重构，创建了 progress.go，是否运行测试？

⏵ 是的，运行测试

  → test_run (package: internal/agent)

🤖 所有测试通过！是否提交代码？
```

### 场景 3: 批量操作
```bash
⏵ 检查所有 namespace 的资源配额使用情况，生成报告

✻ Thinking…
  → kubectl_get_namespaces
  → kubectl_get_resourcequotas (循环所有 namespace)
  → file_write (生成 resource-quota-report.md)

🤖 已生成报告，保存在 resource-quota-report.md
   发现 3 个 namespace 超过 80% 配额，需要关注
```

## 技术选型

- **终端 UI**: readline + golang.org/x/term
- **进度显示**: 自定义动画 + ANSI 控制码
- **流式输出**: SSE / WebSocket
- **会话存储**: SQLite / JSON
- **代码分析**: Go AST / 正则

## 性能优化

1. **上下文智能裁剪**: 只保留最相关的历史
2. **工具结果压缩**: 大输出自动摘要
3. **并发执行**: 独立工具并行调用
4. **缓存机制**: 重复查询结果缓存
5. **增量更新**: 只更新变化的 UI 部分

## 安全考虑

- 危险操作需要确认
- 文件操作限制在项目目录
- Git 操作需要用户批准
- 敏感信息脱敏显示
