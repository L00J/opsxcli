# 11 - Agent 工程化与反馈回路

> 本文档阐述 Harness Engineering 理论框架，以及 opsxcli 项目如何将该理论落地为工程实践。
> 目标读者：AI Agent 开发者、技术负责人。
> 配套文档：`AGENTS.md`（导航地图）、`ARCHITECTURE.md`（架构地图）、`TASTE.md`（品味不变式）。

---

## 1. 核心定义：Agent = Model + Harness

### 1.1 什么是 Harness

> **Agent = Model + Harness** — LangChain

Harness 是 Agent 中除 Model 本身之外的一切。在编码智能体的有界上下文中，Harness 分为两个层次：

- **内置 Harness（built-in harness）**：编码智能体提供方构建的部分 — system prompt、代码检索机制、编排系统
- **外部 Harness（outer harness）**：用户为自身用例和系统构建的部分 — 工程规范、约束规则、反馈回路

Model 是"大脑"，Harness 是"身体+环境+工具+规则"。大脑决定智能，但身体和环境决定智能能否被正确、可靠、持续地发挥。

### 1.2 两个目标

> 一个构建良好的 Harness 服务于两个目标：
> 1. **提高智能体首次就做对的概率**（feedforward）
> 2. **提供反馈回路，在问题到达人类前尽可能自我纠正**（feedback）
> — Martin Fowler / Birgitta Böckeler

这两个目标分别对应**前馈指导**和**反馈传感器**，是 Harness 的双引擎。

### 1.3 核心理念

> **Humans steer. Agents execute.** — OpenAI

人类的工作重心从编写每一行代码，转向**设计环境、明确意图和构建反馈回路**。Harness 是将"不成文的知识" — 审美直觉、组织记忆、"我们这里不这么做"的默契 — 外化和显式化的系统。

---

## 2. 控制论基础：前馈与反馈

### 2.1 前馈指导 vs 反馈传感器

这是 Harness Engineering 的核心控制框架，源自控制论（cybernetics）。

| 维度 | 前馈指导（Feedforward Guides） | 反馈传感器（Feedback Sensors） |
|------|-------------------------------|-------------------------------|
| **时机** | 智能体行动**之前** | 智能体行动**之后** |
| **目的** | 预期行为并加以引导 | 观察产出并帮助自我纠正 |
| **效果** | 提高首次做对的概率 | 在问题到达人类前自我纠正 |
| **opsxcli 实例** | TASTE.md 品味不变式、架构分层约定 | golangci-lint 检查、测试覆盖率门禁 |

> 仅有前馈，规则永远无法验证是否奏效；仅有反馈，智能体会重复同样的错误。两者缺一不可。

### 2.2 计算型 vs 推理型

每种控制和传感器都有两种执行类型（Martin Fowler）：

| 类型 | 特点 | 例子 |
|------|------|------|
| **计算型（Computational）** | 确定性、快速、CPU 执行 | 测试、Linter、类型检查、结构分析 |
| **推理型（Inferential）** | 语义分析、AI 审查、GPU/NPU 执行 | AI Code Review、LLM-as-Judge |

计算型传感器成本低、速度快，适合每次变更都运行。推理型传感器更强大但更贵、不确定，适合关键节点。

推理型传感器的特别威力：当反馈信号被优化为 LLM 可消费的格式时（如自定义 Linter 错误信息中嵌入修复指令），形成一种**积极的 prompt injection**。

### 2.3 Steering Loop 掌舵循环

Steering Loop 是 Harness Engineering 的元过程 — 人类通过迭代改进 Harness 来引导智能体。

```
┌──────────┐   智能体工作   ┌──────────┐   人类改进   ┌──────────┐
│  Harness  │ ──────────► │ 观察问题  │ ──────────► │ 更新Harness│
│ (当前版本) │              │(多次发生) │             │ (新版本)   │
└──────────┘              └──────────┘             └──────────┘
      ▲                                                    │
      └────────────────────────────────────────────────────┘
                         循环往复
```

**运作机制**：
1. 智能体在当前 Harness 约束下工作
2. 当一个问题**多次发生**时（单次问题不需要调整 Harness）
3. 人类分析根本原因：缺少前馈指导？还是缺少反馈传感器？
4. 改进对应层的控件
5. 循环往复

**利用 AI 改进 Harness**：在 Steering Loop 中，编码智能体可以更便宜地构建自定义控件 — 生成结构测试、从观察模式中草拟规则、搭建自定义 Linter、从代码考古中创建 How-to 指南。

### 2.4 Keep Quality Left — 把质量左移

> 尽早在生产路径中部署检查。越早发现问题，修复成本越低。— Martin Fowler

在变更生命周期中，将反馈传感器按成本和速度分布：

| 阶段 | 应运行的传感器 | 特点 |
|------|---------------|------|
| **提交前** | Linter、快速测试套件、基础代码审查 | 毫秒~秒级，每次必跑 |
| **集成后** | 变异测试、全局代码审查 | 较贵，CI 中运行 |
| **持续监控** | 死代码检测、覆盖率质量分析、依赖扫描 | 独立于变更生命周期 |

---

## 3. 三类调节 Harness

> Agent Harness 如同一个控制论调节器（governor），通过前馈和反馈将代码库调节到期望状态。
> 区分这些类别有助于精确沟通，因为每个类别的可 harness 性和复杂度不同。
> — Martin Fowler / Birgitta Böckeler

### 3.1 可维护性 Harness（Maintainability）

这是目前最成熟的类型。我们拥有大量现成的工具：

- **计算型传感器**可靠地捕获结构问题：重复代码、圈复杂度、缺失测试、架构漂移、风格违规
- **推理型传感器**可部分处理需要语义判断的问题：语义重复代码、冗余测试、暴力修复、过度工程
- **两者的盲区**：误诊问题、过度工程和不必要的功能、被误解的指令 — 这些仍需人类监督

### 3.2 架构适应度 Harness（Architecture Fitness）

定义和检查应用的架构特性（Fitness Functions）：

- 前馈：性能需求描述、编码规范
- 反馈：性能测试验证、日志质量审查

### 3.3 行为 Harness（Behaviour）

这是最大的挑战 — 如何引导和感知应用是否按预期功能运行：

- 前馈：功能规格说明（从简短 prompt 到多文件描述）
- 反馈：AI 生成的测试套件 + 手动测试
- **当前局限**：对 AI 生成测试的信任仍不足，需结合已验证的固定装置模式（approved fixtures pattern）

---

## 4. Harnessability 与设计启示

### 4.1 可 Harness 性

并非所有代码库都同样适合被 Harness：

- 强类型语言天然具备类型检查传感器
- 清晰的模块边界使架构约束规则可行
- 框架（如 Spring）抽象掉细节，隐式提高 Agent 成功率

> **环境亲和性（Ambient Affordances）**：环境本身的结构属性使 Agent 更容易被 Harness — 结构清晰、可导航、可操作。

### 4.2 Ashby 必要多样性定律

> 调节器必须拥有至少与被调节系统相同的多样性。LLM 能产生几乎任何代码，但承诺一个拓扑结构可以缩小这个空间，使全面的 Harness 更可行。

这就是为什么预定义的服务拓扑（Harness 模板）是一种**多样性缩减策略**。

### 4.3 Harness 模板

大多数企业有少数几种服务拓扑覆盖 80% 的需求。这些服务模板可能演变为 Harness 模板 — 一组将编码智能体绑定到特定结构、约定和技术栈的前馈指导和反馈传感器。

---

## 5. 反馈回路设计理论

### 5.1 Ralph Wiggum 循环（多轮自我评审）

> 指示智能体在本地审核自身更改，请求额外审查，对反馈做出响应，循环往复直到所有审查人员满意。— OpenAI

```
第 1 轮：Agent 完成代码修改
    │
第 2 轮：Agent 本地自审（审查自己的 diff）
    │
第 3 轮：运行测试 + Lint（计算型反馈传感器）
    │
第 4 轮：根据结果修复问题
    │
第 5 轮：提交，触发 CI（外部反馈传感器）
    │
第 6 轮：人类审查（仅在需要判断时介入）
    │
合并（所有检查通过）
```

### 5.2 黄金原则

从 OpenAI 的 Codex 实践中提炼的四条核心原则（opsxcli 同样适用）：

1. **优先使用共享工具包**（`internal/logger`、`internal/output`），将不变式集中管理
2. **不使用"YOLO 式"探测数据** — 验证边界，依赖类型化 SDK，不基于猜测构建
3. **统一错误处理** — 所有错误使用 `%w` 包装，Sentinel 错误定义在包级别
4. **禁止全局状态** — `commandRegistry` 和 `providerRegistry` 应封装为结构体

---

## 6. opsxcli 代码对照

本节说明 opsxcli 的实际实现如何体现 Harness Engineering 理论。

### 6.1 前馈指导实例

| 理论概念 | opsxcli 实现 | 代码位置 |
|----------|-------------|----------|
| 品味不变式 | TASTE.md + 命名/错误/依赖规范 | 项目根目录 |
| 计算型前馈 | `.golangci.yml` Linter 规则（20+ linter） | 项目根目录 |
| 架构分层约定 | cmd/ → plugins/ → internal/ 单向依赖 | 全项目 |
| 渐进式披露 | docs/01~10 编号文档体系 | `docs/` |
| Agent 行为引导 | 五层 System Prompt 构建 | `internal/agent/prompt/` |

### 6.2 反馈传感器实例

| 理论概念 | opsxcli 实现 | 当前状态 |
|----------|-------------|----------|
| 计算型反馈 | `make test` + 覆盖率门禁 50% | ✅ 52.7%，2282 测试 |
| 计算型反馈 | `make lint` golangci-lint 检查 | ✅ 配置就绪 |
| 计算型反馈 | `go build` 编译验证 | ✅ 零错误 |
| 结构化反馈 | ReAct 循环中的自适应检查点 | `internal/agent/core/agent.go` |
| 安全反馈 | 三级风险审批（strict/balanced/permissive） | `internal/agent/safety/` |

### 6.3 Steering Loop 实例

opsxcli 的 **Evolver 自我进化引擎**是 Steering Loop 的代码级实现：

```
Evolver 引擎 (internal/agent/evolver/)
    │
    ├── engine.go         PREDICT → OBSERVE → ANALYZE → ADAPT 主循环
    ├── experience.go     经验提取与存储
    ├── factual.go        事实性知识积累
    ├── procedural.go     过程性知识（操作序列）
    ├── environment.go    环境感知与适配
    └── progress.go       进度回调机制
```

每次 Agent 执行任务时，Evolver 在 ReAct 循环前执行 PREDICT 阶段，结果通过 RWMutex 注入后续的 System Prompt。当问题多次发生时，进化结果自动调整后续行为 — 这正是 Steering Loop 的自动化实现。

### 6.4 品味不变式实例

| 类别 | 不变式 | 理由 |
|------|--------|------|
| 语言 | 所有注释和错误信息使用中文 | 面向中文运维团队 |
| 命名 | 导出函数 PascalCase，内部函数 camelCase | Go 标准 |
| 错误 | 使用 `%w` 包装错误，禁止吞掉错误 | 保留错误链 |
| 依赖 | cmd/ → plugins/ → internal/ 单向依赖 | 防止循环依赖 |
| 日志 | 统一使用 `internal/logger`（slog） | 禁止 `fmt.Println` |
| 包名 | 全小写，不含下划线 | Go 惯例 |

---

## 7. CLI 可观测性

> opsxcli 是 CLI 工具 — 用户敲命令拿结果就退出。不需要 Prometheus/OpenTelemetry/Grafana 等服务端可观测性栈。
> CLI 可观测性 = **slog 结构化日志 + 审计日志 + `--debug` 开关**。

### 7.1 结构化日志

opsxcli 使用 `log/slog` 统一日志系统，通过键值对使 Agent 执行过程可追溯：

| 键名 | 用途 | 示例 |
|------|------|------|
| `tool` | 工具名称 | `local_bash`, `ssh_execute` |
| `round` | ReAct 轮次 | `1`, `4`, `16` |
| `provider` | LLM Provider | `openai`, `deepseek` |
| `risk` | 风险等级 | `safe`, `high`, `critical` |
| `session` | 会话 ID | `sess_abc123` |
| `duration` | 执行耗时 | `1.2s`, `150ms` |

| 级别 | 使用场景 | 输出目的地 |
|------|----------|-----------|
| `Debug` | LLM 原始请求/响应、Prompt 内容 | `--debug` 时输出 |
| `Info` | Agent 启动/完成、命令执行 | 始终输出 |
| `Warn` | Token 超限、降级行为 | 始终输出 |
| `Error` | 工具失败、LLM API 错误 | 始终输出 + 审计日志 |

### 7.2 审计日志

安全控制器（`internal/agent/safety/controller.go`）记录每次工具执行的审计日志：

- 路径：`~/.opsxcli/audit/audit.log`
- 内容：工具名称、参数、风险等级、是否批准、执行结果、时间戳、会话 ID
- 历史上限：1000 条（`maxHistorySize`），防止内存无限增长

---

## 8. 参考资源

| 资源 | 说明 |
|------|------|
| [Martin Fowler — Harness Engineering](https://martinfowler.com/articles/harness-engineering.html) | 理论框架，控制论视角，含前馈/反馈/Steering Loop |
| [OpenAI — Harness Engineering](https://openai.com/zh-Hans-CN/index/harness-engineering) | 实践报告，含量化数据和 Codex 实战经验 |
| [LangChain — Agent = Model + Harness](https://blog.langchain.com/the-anatomy-of-an-agent-harness/) | 术语定义 |
| [Anthropic — Effective Harnesses for Long-Running Agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) | 长时间运行 Agent 的 Harness 设计 |
| golangci-lint 文档 | `golangci-lint.run` — Linter 配置参考 |
| Effective Go | `go.dev/doc/effective_go` — Go 代码规范 |

---

*本文档为 opsxcli 项目第 11 份设计文档，阐述 Harness Engineering 理论框架及其在 opsxcli 中的落地实践。*
*文档版本: v2.0 | 最后更新: 2026-04 | 配套文档: AGENTS.md, ARCHITECTURE.md, TASTE.md*
