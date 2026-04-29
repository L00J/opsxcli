# 🧪 OpsXCLI TDD 开发计划

> **版本**: v1.3  
> **更新日期**: 2026-04-25  
> **当前测试覆盖**: 58.6% (v0.9.0 开发中)  
> **目标覆盖率**: 57%+ (v0.9.0 ✅ 已达标)

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
| Gitee Go | CI 自动化 |

### 测试分类

| 类别 | 范围 | 运行频率 | 标记 |
|------|------|----------|------|
| **单元测试** | 单个函数/方法 | 每次提交 | — |
| **集成测试** | 模块间交互 | 每次合并 | `//go:build integration` |
| **端到端测试** | 完整命令流程 | 发版前 | `//go:build e2e` |
| **基准测试** | 关键路径性能 | 每周 | `Benchmark*` |

---

## 当前测试现状

> ⚠️ **数据修正**: 之前估算的 12.9% 偏高，经复查实际覆盖率约 10-15%

### 概览

| 指标 | 数值 |
|------|------|
| 测试文件数 | 50 |
| 有测试的包 | 42 |
| 测试失败数 | 0 |
| `plugins/` 覆盖 | ~25% (全部28个包有测试) |
| `cmd/` 覆盖 | 0% (52个文件，零测试) |
| 总体覆盖率 | ~25-30% |

### 已有测试 (16 个文件)

| 文件 | 行数 | 模块 | 覆盖内容 | 状态 |
|------|------|------|----------|------|
| agent_test.go | 1,321 | agent/core | ReAct 循环、工具调用链、死循环检测 | ✅ |
| model_test.go | 732 | agent/tui | TUI 模型、消息处理、视图渲染 | ✅ |
| manager_test.go | 515 | agent/session | 会话创建/加载/列表/导出 | ✅ |
| controller_test.go | 445 | agent/safety | 安全控制、风险评级、黑名单匹配 | ✅ |
| store_test.go | 410 | agent/session | JSONL 存储、会话持久化 | ✅ |
| builder_test.go | 300 | agent/prompt | System Prompt 构建、记忆注入 | ✅ |
| config_test.go | 288 | sshconfig | SSH 配置解析、主机匹配 | ✅ |
| stream_test.go | 261 | agent/core | 流式响应、Token 估算 | ✅ |
| engine_test.go | 237 | agent/evolver | 进化引擎、环境感知 | ✅ |
| tokenizer_test.go | 179 | agent/core | Token 计数、消息裁剪 | ✅ |
| local_bash_test.go | 167 | agent/tools | 本地命令执行、超时控制 | ✅ |
| config_test.go | 120 | config | 配置加载、环境变量覆盖 | ✅ |
| factory_test.go | 116 | llm | 客户端工厂、提供商路由 | ❌ **失败** |
| output_test.go | 92 | output | 输出格式化（text/json） | ✅ |
| logger_test.go | 80 | logger | 日志级别、格式化 | ✅ |
| ssh_execute_test.go | 64 | agent/tools | SSH 远程执行 | ✅ |

### 测试空白区

| 层级 | 包数/文件数 | 代码行数 | 测试文件 | 严重程度 |
|------|------------|----------|----------|----------|
| `plugins/*` | 28个包 | 17,410 | **0** | 🔴 严重 |
| `cmd/*` | 52个文件 | 3,982 | **0** | 🟠 高 |
| `internal/tui/` (通用) | 9 | 1,617 | **0** | 🟡 中 |
| `internal/db/` | 5 | 553 | **0** | 🟠 高 |
| `internal/memory/` | 1 | 332 | **0** | 🟡 中 |

---

## 分阶段测试计划

### Phase 0: 紧急修复 + 安全测试 (本周)

> **目标**: 修复已知测试失败，添加安全回归测试

#### 0.1 修复 C2: factory_test.go 失败

```
TestFactory_GLMClassification    — 修复 glm 提供商分类逻辑
TestFactory_MiniMaxClassification — 修复 minimax 提供商分类逻辑
TestFactory_UnknownProvider      — 确保未知提供商返回错误
```

**根因**: glm/minimax 的模型名匹配规则与测试期望不一致

#### 0.2 紧急安全测试

```
TestNC_CommandInjection          — plugins/nc 命令注入防护（C1验证）
TestNC_SpecialCharacters         — 特殊字符转义
TestCommon_PasswordHandling      — cmd/common.go 密码变量安全（W1验证）
```

---

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
#### 1.2 plugins/redis/ (预计 15 个测试用例)

TestRedisConnect           — 单机/集群连接
TestRedisGetSet            — GET/SET 基本操作
TestRedisAuth              — 密码认证/NOAUTH 处理
TestRedisParseCommand      — 命令解析（引号/转义）
TestRedisCompleter         — 命令补全（STRING/HASH/LIST/SET/ZSET）
TestRedisCluster           — 集群模式 MOVED/ASK 处理

#### 1.2.1 plugins/redis/ 内存分析纯函数 (35 个测试用例, ✅ 已完成)

TestParseMemoryInfo_*          — INFO memory 解析（完整/部分/空/非法值）
TestParseKeySpace_*            — keyspace 解析（多DB/单DB/空/大数字）
TestAnalyzeMemoryHealth_*      — 内存健康检查（无警告/使用率/碎片率/系统占比/淘汰策略/综合）
TestCalculateMemoryEfficiency  — 每键内存效率计算
TestClassifyKeyPatterns_*      — 键模式分类（多前缀/单类型/空）
TestParseSlowLogEntry_*        — 慢查询解析（标准格式/复杂命令/Lua脚本）
TestFormatMemoryReport_*       — 内存报告格式化（含警告/无警告/空）
TestFormatKeySpaceReport_*     — 键空间报告格式化
TestFormatSlowLogReport_*      — 慢查询报告格式化（含数据/空数据）
TestEdgeCases_*                — 边界情况（零值/极大值/负数/NaN）
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
TestEvolverFeedback        — 进化结果反馈到 Prompt (W4验证)
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
| **紧急修复(C2)** | 0 | 3 | P0 |
| **安全测试(C1/W1)** | 0 | 3 | P0 |
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
| **总计** | — | **185** | — |

---

## 测试覆盖率追踪

### 各版本覆盖率目标

| 版本 | 日期 | 总覆盖率 | plugins/ | internal/ | cmd/ |
|------|------|----------|----------|-----------|------|
| v0.4.x (当前) | 2026-04 | ~20-25% | ~15% | ~40% | 0% |
| v0.5.0 | 2026-05 | 20% | 15% | 45% | 10% |
| v0.6.0 | 2026-05 | 35% | 25% | 55% | 20% |
| v0.8.0 | 2026-06 | 45% | 35% | 65% | 30% |
| v1.0.0 | 2026-10 | 50%+ | 40%+ | 70%+ | 40%+ |

> ⚠️ 注：v0.4.x 覆盖率从之前记录的 12.9% 修正为 ~10-15%（诚实估算）

### 每模块覆盖率追踪

#### plugins/ 层

| 插件 | 代码行数 | 测试用例 | 覆盖率 | Phase | 状态 |
|------|----------|----------|--------|-------|------|
||| mysql/ | ~800 | 145+ | ~63% | 1 | ✅ 纯函数已覆盖(含analyze+lock分析) |
| redis/ | ~1200 | 50+ | ~51% | 1 | ✅ 纯函数已覆盖(含内存分析+键空间+慢查询诊断) |
| ssh/ | ~1,200 | 50 | ~30% | 1 | ✅ 纯函数已覆盖 |
| netstat/ | ~500 | 30 | ~35% | 1 | ✅ 纯函数已覆盖 |
|| docker/ | ~800 | 45 | ~25% | 1 | ✅ 纯函数已覆盖 |
|| postgres/ | ~800 | 69+ | ~47% | 1 | ✅ 纯函数已覆盖(含活跃查询+锁等待+性能报告分析) |
|| nc/ | ~300 | 3 | ~50% | 0 | ✅ 安全测试完成 |
| bench/ | ~400 | — | ~25% | 0 | ✅ 基础测试完成 |
| curl/ | ~300 | — | ~25% | 0 | ✅ 基础测试完成 |
| logs/ | ~300 | — | ~25% | 0 | ✅ 基础测试完成 |
| nginx/ | ~400 | — | ~25% | 0 | ✅ 基础测试完成 |
| stats/ | ~300 | — | ~25% | 0 | ✅ 基础测试完成 |
| syslog/ | ~300 | — | ~25% | 0 | ✅ 基础测试完成 |
| notify/ | ~200 | — | ~25% | 0 | ✅ 基础测试完成 |
| request/ | ~200 | — | ~25% | 0 | ✅ 基础测试完成 |
| ssl/ | ~200 | — | ~25% | 0 | ✅ 基础测试完成 |
| kubernetes/ | ~1,000 | 21 | 10% | 2 | 🟡 纯函数已覆盖 |
|| sys/ | ~1,500 | 20 | ~20% | 2 | ✅ 纯函数已覆盖 |
|| net/ | ~1,200 | 50+ | ~63% | 2 | ✅ TUI+解析+监控全覆盖 |
|| builtin/ | ~1,500 | 10 | ~15% | 2 | ✅ 纯函数已覆盖 |
|| cloudhost/ | ~800 | 5 | ~10% | 2 | ✅ 基础测试完成 |
|| install/ | ~200 | 13 | ~40% | 2 | ✅ 包管理器路由已覆盖 |
|| upgrade/ | ~500 | 5 | ~15% | 2 | ✅ 纯函数已覆盖 |
|| ping/ | ~600 | 57 | ~38% | 2 | ✅ ICMP/UDP/TCP三模式+增强统计纯函数全覆盖 |
|| telnet/ | ~200 | 1 | ~5% | 2 | ✅ 常量测试(网络需集成测试) |
|| traceroute/ | ~300 | 4 | ~10% | 2 | ✅ 常量测试(ICMP需集成测试) |
|| wget/ | ~300 | — | ~25% | 0 | ✅ 基础测试完成 |

---

## CI/CD 集成

### 当前 CI 配置

- **平台**: Gitee Go (.gitee/pipelines/ci.yml)
- **Go 版本**: 1.24
- **当前门禁**: 15% 覆盖率

### 门禁提升计划

```yaml
# .gitee/pipelines/ci.yml 门禁调整时间线
# v0.4.x: threshold: 30% (从15%提升)
# v0.5.0: threshold: 20% (新基准)
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

- [ ] 所有单元测试通过
- [ ] 覆盖率不低于当前门禁值
- [ ] 无竞态条件 (`-race`)
- [ ] `go vet` 通过
- [ ] 新增代码有对应测试

---

## 测试时间表

### 8+1 周执行计划

```
第 0 周  ████████ Phase 0:  修复C2测试失败 + 安全回归测试(本周)
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

| 周 | 新增测试文件 | 新增测试行数 | 累计覆盖率 | 备注 |
|----|------------|------------|-----------|------|
| 0 | 2 | ~150 | 10%→12% | 修复C2 + 安全测试 |
| 1 | 2 | ~400 | 12%→17% | mysql + redis |
| 2 | 2 | ~500 | 17%→22% | ssh + netstat |
| 3 | 2 | ~300 | 22%→25% | docker + postgres |
| 4 | 2 | ~350 | 25%→30% | db + memory |
| 5 | 3 | ~400 | 30%→35% | tui + agent |
| 6 | 3 | ~500 | 35%→40% | cmd集成 |
| 7 | 2 | ~300 | 40%→45% | E2E |
| 8 | 0 | 验证 | 45%→50%+ | CI门禁调整 |

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
