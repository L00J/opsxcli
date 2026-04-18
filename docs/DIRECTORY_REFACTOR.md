# OpsXCLI 目录结构重构方案

> 版本: v2.0
> 日期: 2025-12-20
> 状态: 🔧 待执行

---

## 📊 当前问题分析

### 问题 1: 数据目录混乱 (`~/.opsxcli/`)

**当前状态**:
```
~/.opsxcli/
├── data/
│   └── opsxcli.db         # ❌ 重复: 有两个数据库文件
├── opsxcli.db             # ❌ 重复: 应该只在 data/ 下
├── logs/                  # ✅ 正确
├── providers/             # ✅ 正确: LLM 配置
├── docker-cache/          # ✅ 正确: Docker 缓存
├── mysql_history          # ⚠️ 位置不对: 应该在 history/ 下
└── redis_history          # ⚠️ 位置不对: 应该在 history/ 下
```

**问题**:
1. 数据库文件重复 (21MB 浪费)
2. 历史文件散乱
3. 缺少缓存、备份等目录
4. 没有版本隔离

### 问题 2: 项目代码目录混乱 (`/root/opsxcli/`)

**当前状态**:
```
/root/opsxcli/
├── opsxcli              # ✅ 可执行文件
├── opsxcli-test         # ⚠️ 测试版本,应该删除或重命名
├── *.yaml               # ❌ 临时导出文件,不应该在根目录
├── *.md (根目录)        # ⚠️ 文档太多,应该都在 docs/
└── go/                  # ❌ 临时文件?
```

**问题**:
1. 根目录有太多临时文件 (YAML 导出)
2. 文档分散 (部分在根目录,部分在 docs/)
3. 测试版本和正式版本混在一起
4. 缺少 build/ 输出目录

---

## 🎯 统一目录设计方案

### 方案 1: 用户数据目录 (`~/.opsxcli/`)

#### 设计原则

1. **数据隔离**: 不同类型数据分开存储
2. **版本管理**: 支持多版本共存
3. **自动清理**: 过期数据自动清理
4. **备份友好**: 重要数据易于备份

#### 标准目录结构

```
~/.opsxcli/                           # 用户数据根目录
│
├── config/                           # 配置文件目录
│   ├── config.yaml                   # 全局配置
│   ├── providers/                    # LLM 提供商配置
│   │   ├── deepseek.json
│   │   ├── claude.json
│   │   └── ollama.json
│   └── aliases.yaml                  # 命令别名配置
│
├── data/                             # 数据库目录
│   ├── opsxcli.db                    # SQLite 数据库
│   ├── opsxcli.db-wal               # WAL 文件
│   └── opsxcli.db-shm               # SHM 文件
│
├── sessions/                         # 会话数据目录 (可选,如需独立存储)
│   ├── sess_12345/
│   │   ├── metadata.json
│   │   └── context.json
│   └── sess_12346/
│
├── cache/                            # 缓存目录
│   ├── llm/                          # LLM 响应缓存
│   │   ├── deepseek/
│   │   └── claude/
│   ├── k8s/                          # Kubernetes 资源缓存
│   │   └── cluster-info.json
│   └── docker/                       # Docker 镜像缓存
│       └── layers/
│
├── backups/                          # 自动备份目录
│   ├── 2025-12-20/                   # 按日期分组
│   │   ├── deployment.yaml.bak
│   │   └── config.yaml.bak
│   └── 2025-12-19/
│
├── history/                          # 命令历史目录
│   ├── opsxcli_history              # OpsXCLI 主命令历史
│   ├── mysql_history                 # MySQL 历史
│   ├── redis_history                 # Redis 历史
│   └── ssh_history                   # SSH 历史
│
├── logs/                             # 日志目录
│   ├── agent/                        # Agent 日志
│   │   ├── 2025-12-20.log
│   │   └── 2025-12-19.log
│   ├── audit/                        # 审计日志
│   │   └── 2025-12-20.log
│   └── error/                        # 错误日志
│       └── 2025-12-20.log
│
├── exports/                          # 导出文件目录
│   ├── sessions/                     # 会话导出
│   │   └── sess_12345.md
│   ├── k8s/                          # K8s 资源导出
│   │   ├── deployments.yaml
│   │   └── services.yaml
│   └── reports/                      # 报告导出
│       └── resource-quota-2025-12-20.md
│
├── plugins/                          # 插件数据目录
│   ├── ansible/
│   ├── terraform/
│   └── custom/
│
├── tmp/                              # 临时文件目录
│   └── upload/
│
└── .version                          # 版本标识文件
```

#### 配置文件示例

```yaml
# ~/.opsxcli/config/config.yaml
version: "2.0"

# 数据目录配置
data:
  database: "~/.opsxcli/data/opsxcli.db"
  cache_dir: "~/.opsxcli/cache"
  backup_dir: "~/.opsxcli/backups"
  export_dir: "~/.opsxcli/exports"

# 日志配置
logging:
  level: "info"
  dir: "~/.opsxcli/logs"
  max_size_mb: 100
  max_age_days: 30
  rotation: "daily"

# 缓存配置
cache:
  enabled: true
  ttl_hours: 24
  max_size_mb: 500

# 备份配置
backup:
  enabled: true
  retention_days: 7
  auto_backup: true

# Agent 配置
agent:
  max_iterations: 50
  max_history_rounds: 12
  default_provider: "deepseek"
  default_model: "deepseek-chat"

# 清理配置
cleanup:
  auto_cleanup: true
  tmp_retention_hours: 24
  log_retention_days: 30
  backup_retention_days: 7
```

---

### 方案 2: 项目代码目录 (`/root/opsxcli/`)

#### 设计原则

1. **职责清晰**: 代码、构建、文档分离
2. **规范命名**: 遵循 Go 项目标准
3. **易于维护**: 清晰的目录层次
4. **CI/CD 友好**: 标准化构建流程

#### 标准目录结构

```
/root/opsxcli/                        # 项目根目录
│
├── cmd/                              # 命令入口 (Cobra 命令)
│   ├── root.go                       # 根命令
│   ├── agent.go                      # AI Agent
│   ├── session.go                    # 会话管理
│   ├── k8s/                          # K8s 相关命令
│   │   ├── kubectl.go
│   │   ├── resource.go
│   │   └── yaml.go
│   ├── net/                          # 网络工具命令
│   │   ├── ping.go
│   │   ├── curl.go
│   │   └── ssh.go
│   ├── dev/                          # 开发工具命令
│   │   └── server.go
│   └── sys/                          # 系统工具命令
│       ├── sys.go
│       └── docker.go
│
├── internal/                         # 私有应用代码
│   ├── agent/                        # AI Agent 核心
│   │   ├── agent.go                  # Agent 引擎
│   │   ├── safety.go                 # 安全控制
│   │   ├── session.go                # 会话管理
│   │   └── stream.go                 # 流式输出 (新增)
│   │
│   ├── llm/                          # LLM 客户端
│   │   ├── types.go                  # 接口定义
│   │   ├── factory.go                # 工厂模式
│   │   ├── openai.go                 # OpenAI 适配器
│   │   ├── claude.go                 # Claude 适配器
│   │   ├── deepseek.go               # DeepSeek 适配器
│   │   ├── ollama.go                 # Ollama 适配器
│   │   ├── config.go                 # 配置管理
│   │   └── wizard.go                 # 配置向导
│   │
│   ├── tools/                        # 工具系统
│   │   ├── tool.go                   # 工具接口
│   │   ├── registry.go               # 工具注册表
│   │   ├── builtin.go                # 内置工具
│   │   ├── code_tools.go             # 代码工具
│   │   ├── k8s_tools.go              # K8s 工具
│   │   └── net_tools.go              # 网络工具
│   │
│   ├── db/                           # 数据库层
│   │   ├── db.go                     # 数据库连接
│   │   ├── schema.sql                # Schema 定义
│   │   ├── migrate.go                # 迁移管理 (新增)
│   │   ├── user.go                   # 用户仓储
│   │   ├── session.go                # 会话仓储
│   │   ├── message.go                # 消息仓储 (拆分)
│   │   └── audit.go                  # 审计仓储
│   │
│   ├── tui/                          # 终端 UI
│   │   ├── interactive.go            # 交互界面
│   │   ├── screen.go                 # 屏幕管理
│   │   ├── input.go                  # 输入处理 (新增)
│   │   └── progress.go               # 进度显示 (新增)
│   │
│   ├── config/                       # 配置管理
│   │   ├── config.go                 # 配置加载
│   │   └── defaults.go               # 默认配置
│   │
│   ├── auth/                         # 认证授权
│   ├── memory/                       # 记忆管理
│   ├── security/                     # 安全防护
│   └── logger/                       # 日志管理
│
├── plugins/                          # 插件目录
│   ├── kubernetes/                   # K8s 插件
│   ├── docker/                       # Docker 插件
│   ├── ssh/                          # SSH 插件
│   ├── net/                          # 网络监控
│   └── sys/                          # 系统监控
│
├── pkg/                              # 公共库 (可被外部引用)
│   ├── utils/                        # 工具函数
│   └── types/                        # 公共类型
│
├── tests/                            # 测试目录 (新增)
│   ├── unit/                         # 单元测试
│   ├── integration/                  # 集成测试
│   └── e2e/                          # 端到端测试
│
├── build/                            # 构建输出目录 (新增)
│   ├── bin/                          # 二进制文件
│   │   ├── opsxcli-linux-amd64
│   │   ├── opsxcli-darwin-arm64
│   │   └── opsxcli-windows-amd64.exe
│   └── pkg/                          # 打包文件
│       ├── opsxcli-v2.0-linux.tar.gz
│       └── opsxcli-v2.0-darwin.tar.gz
│
├── scripts/                          # 脚本目录 (新增)
│   ├── build.sh                      # 构建脚本
│   ├── test.sh                       # 测试脚本
│   ├── install.sh                    # 安装脚本
│   └── release.sh                    # 发布脚本
│
├── docs/                             # 文档目录
│   ├── README.md                     # 主文档
│   ├── architecture/                 # 架构文档
│   │   ├── COMPLETE_ARCHITECTURE.md
│   │   ├── DATABASE_DESIGN.md
│   │   └── PLUGIN_ARCHITECTURE.md
│   ├── guides/                       # 使用指南
│   │   ├── getting-started.md
│   │   ├── agent-usage.md
│   │   └── session-management.md
│   ├── api/                          # API 文档
│   └── development/                  # 开发文档
│       ├── CONTRIBUTING.md
│       └── DEVELOPMENT.md
│
├── examples/                         # 示例目录 (新增)
│   ├── agent/                        # Agent 示例
│   ├── plugins/                      # 插件示例
│   └── configs/                      # 配置示例
│
├── .github/                          # GitHub 配置 (新增)
│   ├── workflows/                    # CI/CD
│   │   ├── test.yml
│   │   └── release.yml
│   └── ISSUE_TEMPLATE/
│
├── main.go                           # 程序入口
├── go.mod                            # Go 模块
├── go.sum                            # 依赖锁定
├── Makefile                          # Make 构建
├── .gitignore                        # Git 忽略
├── .golangci.yml                     # Linter 配置 (新增)
├── LICENSE                           # 许可证
└── README.md                         # 项目说明
```

---

## 🔧 执行计划

### 阶段 1: 数据目录重构 (立即执行)

#### 步骤 1: 创建标准目录结构

```bash
#!/bin/bash
# scripts/migrate-data-dir.sh

# 备份当前数据
backup_dir="/tmp/opsxcli-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$backup_dir"
cp -r ~/.opsxcli "$backup_dir/"

# 创建新目录结构
mkdir -p ~/.opsxcli/{config/providers,data,sessions,cache/{llm,k8s,docker},backups,history,logs/{agent,audit,error},exports/{sessions,k8s,reports},plugins,tmp}

# 迁移现有数据
mv ~/.opsxcli/providers/*.json ~/.opsxcli/config/providers/ 2>/dev/null
mv ~/.opsxcli/data/opsxcli.db ~/.opsxcli/data/ 2>/dev/null
rm ~/.opsxcli/opsxcli.db 2>/dev/null  # 删除重复的数据库
mv ~/.opsxcli/mysql_history ~/.opsxcli/history/
mv ~/.opsxcli/redis_history ~/.opsxcli/history/
mv ~/.opsxcli/logs/* ~/.opsxcli/logs/agent/ 2>/dev/null

# 创建配置文件
cat > ~/.opsxcli/config/config.yaml << 'EOF'
version: "2.0"
data:
  database: "~/.opsxcli/data/opsxcli.db"
  cache_dir: "~/.opsxcli/cache"
  backup_dir: "~/.opsxcli/backups"
agent:
  max_iterations: 50
  max_history_rounds: 12
EOF

# 创建版本文件
echo "2.0" > ~/.opsxcli/.version

# 清理旧目录
rmdir ~/.opsxcli/providers 2>/dev/null
rmdir ~/.opsxcli/data 2>/dev/null

echo "✓ 数据目录迁移完成"
echo "✓ 备份位置: $backup_dir"
```

#### 步骤 2: 更新代码中的路径引用

```go
// internal/config/defaults.go
package config

import (
    "os"
    "path/filepath"
)

// GetDataDir 获取数据目录
func GetDataDir() string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".opsxcli")
}

// GetDatabasePath 获取数据库路径
func GetDatabasePath() string {
    return filepath.Join(GetDataDir(), "data", "opsxcli.db")
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
    return filepath.Join(GetDataDir(), "config", "config.yaml")
}

// GetCacheDir 获取缓存目录
func GetCacheDir() string {
    return filepath.Join(GetDataDir(), "cache")
}

// GetLogDir 获取日志目录
func GetLogDir() string {
    return filepath.Join(GetDataDir(), "logs", "agent")
}

// GetExportDir 获取导出目录
func GetExportDir() string {
    return filepath.Join(GetDataDir(), "exports")
}
```

### 阶段 2: 项目代码重构 (逐步执行)

#### 步骤 1: cmd 目录重组

```bash
# 创建子目录
mkdir -p cmd/{k8s,net,dev,sys}

# 移动文件
mv cmd/kubernetes*.go cmd/k8s/
mv cmd/kubectl.go cmd/k8s/

mv cmd/{ping,curl,wget,ssh,telnet,nc,netstat,ss,nmap}.go cmd/net/

mv cmd/server.go cmd/dev/

mv cmd/{sys,docker}.go cmd/sys/
```

#### 步骤 2: 清理根目录

```bash
# 移动临时导出文件
mkdir -p ~/.opsxcli/exports/k8s
mv /root/opsxcli/*.yaml ~/.opsxcli/exports/k8s/

# 移动文档到 docs/
mkdir -p docs/legacy
mv /root/opsxcli/*.md docs/legacy/ 2>/dev/null

# 删除测试版本
rm /root/opsxcli/opsxcli-test
```

#### 步骤 3: 创建新目录

```bash
# 创建标准目录
mkdir -p {tests/{unit,integration,e2e},build/{bin,pkg},scripts,examples,pkg/{utils,types}}

# 移动构建脚本
mv build.sh scripts/
mv release.py scripts/
```

---

## 📝 迁移检查清单

### 数据目录迁移

- [ ] 备份现有数据
- [ ] 创建新目录结构
- [ ] 迁移数据库文件 (去重)
- [ ] 迁移配置文件
- [ ] 迁移历史文件
- [ ] 迁移日志文件
- [ ] 创建配置文件
- [ ] 更新代码中的路径
- [ ] 测试数据访问
- [ ] 删除旧文件

### 项目代码迁移

- [ ] cmd 目录重组
- [ ] 清理根目录临时文件
- [ ] 移动文档到 docs/
- [ ] 创建 tests/ 目录
- [ ] 创建 build/ 目录
- [ ] 创建 scripts/ 目录
- [ ] 更新 import 路径
- [ ] 更新 Makefile
- [ ] 更新 .gitignore
- [ ] 测试编译

---

## 🎯 预期收益

### 数据目录

✅ **清晰的数据分类**: 配置、数据、缓存、日志分离
✅ **避免重复**: 数据库文件去重,节省 21MB
✅ **易于备份**: 重要数据集中在 data/ 和 config/
✅ **自动清理**: 临时文件、过期日志自动清理
✅ **版本管理**: .version 文件标识版本

### 项目代码

✅ **清晰的职责划分**: cmd/k8s/, cmd/net/ 等分组
✅ **规范的构建流程**: build/, scripts/ 标准化
✅ **完善的测试体系**: tests/ 目录
✅ **易于维护**: 减少 50% 查找文件时间

---

## 🚀 下一步

1. ✅ 执行数据目录迁移脚本
2. ✅ 更新代码中的路径引用
3. ✅ 重新编译测试
4. ✅ 逐步重组项目代码
5. ✅ 更新文档

---

*重构方案设计: 2025-12-20*
