# OpsX CLI 项目开发规则

## 项目架构

### 目录结构
```
opsxcli/
├── cmd/                    # 命令入口层
│   ├── agent.go           # AI 助手命令
│   ├── session.go         # 会话管理命令
│   ├── kubectl.go         # Kubernetes 工具
│   └── ...                # 其他工具命令
│
├── internal/              # 内部核心模块（不可外部导入）
│   ├── agent/            # AI 引擎核心（个人开发，不提交）
│   ├── llm/              # 大语言模型客户端
│   ├── tools/            # 工具注册表
│   ├── db/               # 数据库和配置
│   └── auth/             # 认证授权
│
└── plugins/              # 功能插件（可独立使用）
    ├── redis/            # Redis 客户端
    ├── kubernetes/       # Kubernetes 工具集
    ├── docker/           # Docker 工具
    ├── ssh/              # SSH 客户端
    └── sys/              # 系统监控
```

### 架构分层原则

1. **internal/agent/** - AI 引擎层
   - 职责：推理、决策、工具编排、安全控制、自我修正
   - 依赖：internal/tools, internal/llm, plugins/*
   - 特点：个人开发代码，不提交到仓库

2. **internal/tools/** - 工具注册层
   - 职责：工具注册、参数验证、工具发现
   - 依赖：plugins/*
   - 特点：作为 agent 和 plugins 的桥梁

3. **plugins/** - 功能插件层
   - 职责：具体功能实现（Redis、K8s、Docker 等）
   - 依赖：internal/db, internal/logger
   - 特点：可独立使用，也可被 Agent 调用

## 开发规范

### 代码风格

1. **始终使用中文注释和文档**
   ```go
   // ✅ 正确：创建 Redis 客户端
   func GetClient() {}

   // ❌ 错误：Create redis client
   func GetClient() {}
   ```

2. **错误处理规范**
   ```go
   // ✅ 使用中文错误信息
   return fmt.Errorf("连接 Redis 失败: %w", err)

   // ❌ 不要用英文
   return fmt.Errorf("failed to connect: %w", err)
   ```

3. **函数命名**
   - 导出函数：使用 PascalCase（GetClient, Interactive）
   - 内部函数：使用 camelCase（getRedisCompleter, formatOutput）

### 目录职责

1. **cmd/** - 只负责命令行参数解析和调用
   - 不要在这里写业务逻辑
   - 保持简单，只做参数传递

2. **internal/agent/** - AI 引擎核心
   - 模块化设计：agent.go, executor.go, prompt.go 等
   - 支持多种 LLM：DeepSeek, Claude, Ollama
   - 安全控制：高风险操作需确认

3. **plugins/** - 独立功能插件
   - 每个插件都应该能独立运行
   - 不要依赖 agent 或其他插件
   - 提供清晰的函数接口

### 依赖管理

```
✅ 允许的依赖方向：
cmd/ → internal/agent/ → internal/tools/ → plugins/
cmd/ → plugins/

❌ 禁止的依赖方向：
plugins/ → internal/agent/
internal/tools/ → cmd/
```

## 特殊说明

### internal/agent/ 目录
- **个人开发代码**，已添加到 .gitignore
- 包含 AI 引擎的核心实现
- 不提交到公共仓库

### 代码提交前检查
1. 确保 internal/agent/ 不会被提交
2. 所有注释和错误信息都是中文
3. 运行 `go mod tidy` 清理依赖
4. 运行 `go build` 确保编译通过

## 工具开发指南

### 添加新的插件工具

1. 在 `plugins/` 下创建新目录
   ```bash
   mkdir plugins/mytool
   ```

2. 实现核心功能
   ```go
   package mytool

   // Execute 执行工具功能
   func Execute(args map[string]interface{}) (string, error) {
       // 实现逻辑
       return result, nil
   }
   ```

3. 在 `internal/tools/registry.go` 注册工具
   ```go
   registry.Register("mytool", tools.Tool{
       Name:        "mytool",
       Description: "工具描述",
       Parameters:  /* JSON Schema */,
       Function:    mytool.Execute,
   })
   ```

4. 在 `cmd/` 添加命令入口（如果需要独立使用）
   ```go
   // cmd/mytool.go
   var mytoolCmd = &cobra.Command{
       Use:   "mytool",
       Short: "我的工具",
       Run: func(cmd *cobra.Command, args []string) {
           result, err := mytool.Execute(params)
           // 处理结果
       },
   }
   ```

### Agent 可用的工具

Agent 可以调用所有已注册的工具，包括：
- **系统工具**：shell_command, file_read, file_write
- **网络工具**：ping, curl, wget, telnet
- **容器工具**：kubectl, docker
- **数据库工具**：redis, mysql, postgres
- **监控工具**：sys_monitor, net_monitor

## 最佳实践

1. **保持插件独立性**
   - 每个插件应该能单独使用
   - 不要强依赖 Agent 功能

2. **使用统一的配置管理**
   - 通过 `internal/db` 管理常用连接配置
   - 支持默认值和环境变量

3. **日志规范**
   - 使用 `internal/logger` 统一日志
   - 分级：Debug, Info, Success, Warning, Error

4. **安全控制**
   - 高风险操作需要确认
   - 记录审计日志
   - 敏感信息不要打印

## 测试规范

1. **单元测试**
   ```bash
   go test ./plugins/mytool/...
   ```

2. **集成测试**
   ```bash
   go test ./internal/agent/...
   ```

3. **手动测试**
   ```bash
   go build && ./opsxcli agent "测试任务"
   ```

## 参考文档

- 查看 `internal/agent/README.md` 了解 AI 引擎设计
- 查看 `docs/` 目录了解详细文档
- 参考现有插件实现新功能
