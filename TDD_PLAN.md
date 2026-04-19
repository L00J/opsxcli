# 🧪 OpsXCLI TDD 开发计划

> **版本**: v1.0  
> **更新日期**: 2026-04-19  
> **当前测试覆盖**: 12.9% (5,327 行 / 41,418 行)  
> **目标覆盖率**: 50%+

---

## 📋 目录

- [测试策略](#测试策略)
- [当前测试现状](#当前测试现状)
- [分阶段测试计划](#分阶段测试计划)
- [测试用例清单](#测试用例清单)
- [测试覆盖率追踪](#测试覆盖率追踪)
- [CI/CD 集成](#cicd-集成)
- [测试时间表](#测试时间表)

---

## 测试策略

### 总体原则

1. **测试金字塔**: 单元测试(70%) > 集成测试(20%) > 端到端测试(10%)
2. **TDD 流程**: Red → Green → Refactor，先写测试再写实现
3. **每次提交前运行**: `go test ./...` 确保不引入回归
4. **增量提升**: 不追求一步到位，每个版本提升 10-15%

### 测试工具链

| 工具 | 用途 |
|------|------|
| `go test` | 标准测试框架 |
| `testify/assert` | 断言库（项目已使用） |
| `golangci-lint` | 静态分析 |
| `go test -cover` | 覆盖率统计 |
| `go test -race` | 竞态检测 |
| GitHub Actions | CI 自动化 |

### 测试分类

| 类别 | 范围 | 运行频率 | 标记 |
|------|------|----------|------|
| **单元测试** | 单个函数/方法 | 每次提交 | — |
| **集成测试** | 模块间交互 | 每次合并 | `//go:build integration` |
| **端到端测试** | 完整命令流程 | 发版前 | `//go:build e2e` |
| **基准测试** | 关键路径性能 | 每周 | `Benchmark*` |

---

## 当前测试现状

### 已有测试 (16 个文件, 5,327 行)

| 文件 | 行数 | 模块 | 覆盖内容 |
|------|------|------|----------|
| agent_test.go | 1,321 | agent/core | ReAct 循环、工具调用链、死循环检测 |
| model_test.go | 732 | agent/tui | TUI 模型、消息处理、视图渲染 |
| manager_test.go | 515 | agent/session | 会话创建/加载/列表/导出 |
| controller_test.go | 445 | agent/safety | 安全控制、风险评级、黑名单匹配 |
| store_test.go | 410 | agent/session | JSONL 存储、会话持久化 |
| builder_test.go | 300 | agent/prompt | System Prompt 构建、记忆注入 |
| config_test.go | 288 | sshconfig | SSH 配置解析、主机匹配 |
| stream_test.go | 261 | agent/core | 流式响应、Token 估算 |
| engine_test.go | 237 | agent/evolver | 进化引擎、环境感知 |
| tokenizer_test.go | 179 | agent/core | Token 计数、消息裁剪 |
| local_bash_test.go | 167 | agent/tools | 本地命令执行、超时控制 |
| config_test.go | 120 | config | 配置加载、环境变量覆盖 |
| factory_test.go | 116 | llm | 客户端工厂、提供商路由 |
| output_test.go | 92 | output | 输出格式化（text/json） |
| logger_test.go | 80 | logger | 日志级别、格式化 |
| ssh_execute_test.go | 64 | agent/tools | SSH 远程执行 |

### 测试空白区

| 层级 | 文件数 | 代码行数 | 测试文件 | 严重程度 |
|------|--------|----------|----------|----------|
| `plugins/*` | 69 | 17,410 | **0** | 🔴 严重 |
| `cmd/*` | 43 | 3,982 | **0** | 🟠 高 |
| `internal/tui/` (通用) | 9 | 1,617 | **0** | 🟡 中 |
| `internal/db/` | 5 | 553 | **0** | 🟠 高 |
| `internal/memory/` | 1 | 332 | **0** | 🟡 中 |

---

## 分阶段测试计划

### Phase 1: plugins/ 基础覆盖 (第 1-2 周)

> **目标**: 从 0% → 20% 覆盖率，优先核心插件

#### 1.1 plugins/mysql/ (预计 15 个测试用例)

```
TestMySQLConnect           — 连接成功/失败/超时
TestMySQLConnectAuth       — 密码错误/ASK 模式
TestMySQLExecute           — SQL 执行与结果解析
TestMySQLQueryTable        — 表格输出格式化
TestMySQLFormatValue       — 值格式化（NULL/数字/字符串/时间）
TestMySQLInteractive       — 交互式 shell 启动（mock）
```

#### 1.2 plugins/redis/ (预计 15 个测试用例)

```
TestRedisConnect           — 单机/集群连接
TestRedisGetSet            — GET/SET 基本操作
TestRedisAuth              — 密码认证/NOAUTH 处理
TestRedisParseCommand      — 命令解析（引号/转义）
TestRedisCompleter         — 命令补全（STRING/HASH/LIST/SET/ZSET）
TestRedisCluster           — 集群模式 MOVED/ASK 处理
```

#### 1.3 plugins/ssh/ (预计 20 个测试用例)

```
TestSSHConnect             — 密钥/密码认证
TestSSHExecute             — 远程命令执行 + exit status
TestSSHSFTPUpload          — 文件上传（单文件/递归目录）
TestSSHSFTPDownload        — 文件下载
TestSSHForwardLocal        — 本地端口转发
TestSSHForwardRemote       — 远程端口转发
TestSSHForwardSOCKS        — 动态 SOCKS 代理
TestSSHConfig              — SSH config 解析与匹配
```

#### 1.4 plugins/netstat/ (预计 10 个测试用例)

```
TestParseProcNetTCP        — /proc/net/tcp 解析
TestParseProcNetUDP        — /proc/net/udp 解析
TestTimeWaitStats          — TIME_WAIT 统计
TestTopAnalysis            — 目标地址 TOP 分析
TestFormatConnection       — 连接信息格式化
```

#### 1.5 plugins/docker/ (预计 8 个测试用例)

```
TestParseImageName         — 镜像名解析（registry/repo:tag）
TestMirrorSpeed            — 镜像源测速（mock HTTP）
TestConcurrentPull         — 并发拉取控制
TestResumeDownload         — 断点续传
```

### Phase 2: internal/ 深度覆盖 (第 3-4 周)

> **目标**: 20% → 35% 覆盖率

#### 2.1 internal/db/ (预计 12 个测试用例)

```
TestDBOpen                 — SQLite 打开/关闭
TestDBSchema               — 表结构初始化
TestSessionCRUD            — 会话增删改查
TestAuditLog               — 审计日志写入与查询
TestConnectionSave         — 连接配置保存与加载
TestHistoryRecord          — 命令历史记录
```

#### 2.2 internal/memory/ (预计 8 个测试用例)

```
TestMemorySave             — 记忆存储（短/长期）
TestMemoryRetrieve         — 记忆检索与排序
TestMemoryCategorize       — 分类：knowledge/preference/environment
TestMemoryCleanup          — 过期记忆清理
```

#### 2.3 internal/tui/ (预计 10 个测试用例)

```
TestTableRender            — 表格渲染
TestProgressUpdate         — 进度条更新
TestInputHandler           — 输入处理
TestScreenManager          — 屏幕切换
```

#### 2.4 internal/agent/ 补充 (预计 10 个测试用例)

```
TestEvolverFeedback        — 进化结果反馈到 Prompt
TestToolOutputTruncate     — 输出截断逻辑
TestMultiToolChain         — 多工具串联调用
TestConcurrentAgentRun     — 并发 Agent 运行安全
```

### Phase 3: cmd/ 集成测试 (第 5-6 周)

> **目标**: 35% → 45% 覆盖率

#### 3.1 命令注册测试 (预计 10 个测试用例)

```
TestCommandRegistration    — 所有命令已正确注册
TestCommandHelp            — --help 输出完整
TestCategoryGrouping       — 命令分组正确
TestAliasResolution        — 命令别名解析
```

#### 3.2 参数解析测试 (预计 15 个测试用例)

```
TestPreprocessPassword     — -pPASSWORD 预处理
TestDatabaseFlags          — 数据库参数解析
TestAgentFlags             — Agent 模式参数
TestSSHFlags               — SSH 子命令参数
```

### Phase 4: 端到端测试 (第 7-8 周)

> **目标**: 45% → 50%+ 覆盖率

#### 4.1 完整命令流程 (预计 10 个测试用例)

```
TestE2ESSHConnect          — SSH 连接 → 执行 → 退出
TestE2EMySQLQuery          — MySQL 连接 → 查询 → 结果
TestE2ERedisGetSet         — Redis 连接 → SET → GET → 验证
TestE2EDockerPull          — Docker 镜像拉取全流程（mock registry）
TestE2EAgentQuery          — Agent 单次查询全流程（mock LLM）
```

---

## 测试用例清单

### 按模块统计

| 模块 | Phase | 预计用例数 | 优先级 |
|------|-------|-----------|--------|
| plugins/mysql/ | 1 | 15 | P0 |
| plugins/redis/ | 1 | 15 | P0 |
| plugins/ssh/ | 1 | 20 | P0 |
| plugins/netstat/ | 1 | 10 | P0 |
| plugins/docker/ | 1 | 8 | P0 |
| plugins/postgres/ | 1 | 10 | P1 |
| plugins/ping/ | 1 | 5 | P2 |
| plugins/nmap/ | 1 | 5 | P2 |
| plugins/sys/ | 2 | 8 | P2 |
| plugins/net/ | 2 | 8 | P2 |
| internal/db/ | 2 | 12 | P1 |
| internal/memory/ | 2 | 8 | P1 |
| internal/tui/ | 2 | 10 | P2 |
| internal/agent/ | 2 | 10 | P1 |
| cmd/ 集成 | 3 | 25 | P1 |
| E2E 测试 | 4 | 10 | P2 |
| **总计** | — | **179** | — |

---

## 测试覆盖率追踪

### 各版本覆盖率目标

| 版本 | 日期 | 总覆盖率 | plugins/ | internal/ | cmd/ |
|------|------|----------|----------|-----------|------|
| v0.4.x (当前) | 2026-04 | 12.9% | 0% | ~35% | 0% |
| v0.5.0 | 2026-05 | 25% | 15% | 45% | 10% |
| v0.6.0 | 2026-05 | 35% | 25% | 55% | 20% |
| v0.8.0 | 2026-06 | 45% | 35% | 65% | 30% |
| v1.0.0 | 2026-10 | 50%+ | 40%+ | 70%+ | 40%+ |

### 每模块覆盖率追踪

#### plugins/ 层

| 插件 | 代码行数 | 测试用例 | 覆盖率 | Phase | 状态 |
|------|----------|----------|--------|-------|------|
| mysql/ | ~600 | 15 | 目标 40% | 1 | 🔴 待开始 |
| redis/ | ~800 | 15 | 目标 40% | 1 | 🔴 待开始 |
| ssh/ | ~1,200 | 20 | 目标 50% | 1 | 🔴 待开始 |
| netstat/ | ~500 | 10 | 目标 40% | 1 | 🔴 待开始 |
| docker/ | ~800 | 8 | 目标 30% | 1 | 🔴 待开始 |
| postgres/ | ~600 | 10 | 目标 40% | 1 | 🔴 待开始 |
| kubernetes/ | ~1,000 | 8 | 目标 30% | 2 | 🔴 待开始 |
| sys/ | ~1,500 | 8 | 目标 20% | 2 | 🔴 待开始 |
| net/ | ~1,200 | 8 | 目标 20% | 2 | 🔴 待开始 |
| 其他 | ~9,210 | — | — | 3 | 🔴 待开始 |

---

## CI/CD 集成

### 当前 CI 配置

- **平台**: GitHub Actions (.github/workflows/ci.yml)
- **Go 版本**: 1.24
- **当前门禁**: 15% 覆盖率

### 门禁提升计划

```yaml
# .github/workflows/ci.yml 门禁调整时间线
# v0.5.0: threshold: 25%
# v0.6.0: threshold: 35%
# v0.8.0: threshold: 45%
# v1.0.0: threshold: 50%
```

### CI 流水线优化

| 阶段 | 命令 | 条件 |
|------|------|------|
| **构建** | `CGO_ENABLED=0 go build` | 始终运行 |
| **单元测试** | `go test -race -coverprofile=coverage.out ./...` | 始终运行 |
| **覆盖率检查** | `go tool cover -func=coverage.out` | 覆盖率 ≥ 门禁 |
| **竞态检测** | `go test -race ./...` | 始终运行 |
| **Lint** | `golangci-lint run ./...` | continue-on-error |
| **集成测试** | `go test -tags=integration ./...` | 仅 main 分支 |
| **E2E 测试** | `go test -tags=e2e ./...` | 仅发版时 |

### PR 合并条件

- [x] 所有单元测试通过
- [x] 覆盖率不低于当前门禁值
- [x] 无竞态条件 (`-race`)
- [x] `go vet` 通过
- [x] 新增代码有对应测试

---

## 测试时间表

### 8 周执行计划

```
第 1 周  ████████ Phase 1A: plugins/mysql + plugins/redis 测试
第 2 周  ████████ Phase 1B: plugins/ssh + plugins/netstat 测试
第 3 周  ████████ Phase 1C: plugins/docker + plugins/postgres 测试
第 4 周  ████████ Phase 2A: internal/db + internal/memory 测试
第 5 周  ████████ Phase 2B: internal/tui + internal/agent 补充测试
第 6 周  ████████ Phase 3:  cmd/ 集成测试
第 7 周  ████████ Phase 4:  端到端测试
第 8 周  ████████ 收尾:    覆盖率验证 + CI 门禁调整
```

### 每周产出目标

| 周 | 新增测试文件 | 新增测试行数 | 累计覆盖率 |
|----|------------|------------|-----------|
| 1 | 2 | ~400 | 12.9% → 17% |
| 2 | 2 | ~500 | 17% → 22% |
| 3 | 2 | ~300 | 22% → 25% |
| 4 | 2 | ~350 | 25% → 30% |
| 5 | 3 | ~400 | 30% → 35% |
| 6 | 3 | ~500 | 35% → 40% |
| 7 | 2 | ~300 | 40% → 45% |
| 8 | 0 | 验证 | 45% → 50%+ |

### 测试编写规范

```go
// 测试文件命名: <module>_test.go
// 测试函数命名: Test<功能>_<场景>_<预期结果>

func TestMySQLConnect_Timeout_ReturnsError(t *testing.T) {
    // Arrange
    cfg := mysql.Config{
        Host:     "192.0.2.1", // RFC 5737 测试地址
        Port:     3306,
        Timeout:  1 * time.Second,
    }
    
    // Act
    client, err := mysql.NewClient(cfg)
    
    // Assert
    assert.Error(t, err)
    assert.Nil(t, client)
    assert.Contains(t, err.Error(), "超时")
}
```

---

## 附录

### Mock 策略

| 依赖 | Mock 方式 | 工具 |
|------|----------|------|
| MySQL/PostgreSQL | 接口 mock | testify/mock |
| Redis | miniredis 内存实例 | miniredis |
| SSH 连接 | 本地 SSH 服务器 | testcontainers 或 mock |
| HTTP 服务 | httptest.Server | net/http/httptest |
| 文件系统 | t.TempDir() | 标准库 |
| LLM API | httptest.Server | net/http/httptest |

### 相关文档

- [ROADMAP.md](./ROADMAP.md) — 开发路线图
- [PROJECT_EVALUATION.md](./PROJECT_EVALUATION.md) — 项目评估
- [CLAUDE.md](./CLAUDE.md) — 开发规范
- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) — 架构说明
