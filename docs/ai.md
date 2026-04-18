# opsxcli - AI运维助手

## 功能介绍

opsxcli 内置 AI 运维助手,让你用自然语言描述运维任务,AI 会自动调用相关工具帮你解决问题。

**无需子命令,直接使用:**
```bash
opsxcli "查看磁盘使用情况"
```

## 快速开始

### 1. 首次使用

```bash
opsxcli "查看系统信息"
```

首次运行会自动启动配置向导,引导你设置 AI 提供商。

### 2. 选择 AI 提供商

#### 方案一: DeepSeek（推荐,性价比高）

- 访问 https://platform.deepseek.com/ 注册并获取 API Key
- 费用: ¥1/百万tokens（非常便宜）
- 配置向导时选择 DeepSeek 并输入 API Key

#### 方案二: Ollama（完全免费,本地运行）

```bash
# 安装 Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 下载模型
ollama pull qwen2.5:latest

# 配置向导时选择 Ollama
```

#### 方案三: Claude（功能最强）

- 访问 https://console.anthropic.com/ 获取 API Key
- 费用较高,功能最强大
- 配置向导时选择 Claude 并输入 API Key

### 3. 开始使用

```bash
# 使用默认提供商
opsxcli "查看磁盘使用情况"

# 指定提供商
opsxcli -p deepseek "分析系统负载"
opsxcli -p ollama "查看网络连接"
opsxcli -p claude "诊断K8s问题"
```

## 使用示例

### 系统诊断

```bash
# 查看磁盘使用
opsxcli "查看根目录磁盘使用情况"

# 查看系统负载
opsxcli "查看系统CPU和内存使用情况"

# 查找占用资源最多的进程
opsxcli "找出占用内存最多的5个进程"
```

### 网络诊断

```bash
# 查看网络连接
opsxcli "查看所有TCP连接状态"

# 查找端口占用
opsxcli "查找占用80端口的进程"

# 网络连通性测试
opsxcli "测试到百度的网络连通性"
```

### Kubernetes 运维

```bash
# 查看 Pod 状态
opsxcli "查看default命名空间下的所有pods"

# 诊断问题
opsxcli "查看处于Pending状态的pods并分析原因"

# 查看日志
opsxcli "查看nginx pod的最新100行日志"
```

### 日志分析

```bash
# 查看错误日志
opsxcli "查看/var/log/messages中最近的错误日志"

# 统计日志
opsxcli "统计nginx访问日志中访问量最多的IP"
```

### 文件操作

```bash
# 查找文件
opsxcli "查找所有大于100MB的日志文件"

# 查看文件内容
opsxcli "查看nginx配置文件"

# 搜索文本
opsxcli "在当前目录下所有go文件中搜索包含error的行"
```

## 命令参数

```bash
opsxcli [任务描述] [flags]
```

### 参数说明

| 参数 | 简写 | 说明 | 示例 |
|------|------|------|------|
| `--provider` | `-p` | 指定AI提供商 | `-p deepseek` |
| `--interactive` | `-i` | 交互模式 | `-i` |
| `--query` | `-q` | 任务描述 | `-q "查看系统信息"` |
| `--safety` | `-s` | 安全模式 | `-s strict` |

### 安全模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| `strict` | 严格模式,中风险及以上操作需确认 | 生产环境 |
| `balanced` | 平衡模式,高风险操作需确认（默认） | 一般使用 |
| `permissive` | 宽松模式,仅危险操作需确认 | 测试环境 |

示例:
```bash
# 严格模式
opsxcli -s strict "删除临时文件"

# 宽松模式
opsxcli -s permissive "查看日志"
```

## 交互模式

交互模式允许持续对话,AI 可以根据上下文回答问题:

```bash
opsxcli -i
```

示例对话:
```
🤖 opsxcli AI 助手 [deepseek]
输入 'exit' 或 'quit' 退出

👤 您: 查看磁盘使用情况
🤖 AI: [执行 df 命令并显示结果]
根目录使用了45%，还有充足空间...

👤 您: 那内存使用呢
🤖 AI: [执行 free 命令]
内存使用62%，建议关注...

👤 您: exit
👋 再见！
```

## 工作原理

1. **自然语言理解**: AI 分析你的任务描述
2. **工具选择**: 自动选择合适的工具(df, ps, kubectl 等)
3. **执行命令**: 安全地执行相关命令
4. **结果分析**: AI 分析执行结果并给出建议
5. **多轮对话**: 可以根据结果继续追问

## 支持的工具

### 文件操作（安全 🟢）
- `cat`, `grep`, `ls`, `head`, `tail`, `tree`

### 系统信息（安全 🟢）
- `ps`, `top`, `free`, `df`, `du`, `uname`, `hostname`

### 网络工具（安全 🟢）
- `ping`, `ss`, `netstat`, `ifconfig`

### Kubernetes（安全 🟢 / 高风险 🔴）
- `kubectl get`, `kubectl describe`, `kubectl logs`
- `kubectl delete`（高风险,需确认）

### 危险操作（危险 💀）
- `rm`, `dd`（必须手动确认）

## 安全特性

1. **风险分级**: 所有工具按风险等级分类
2. **操作确认**: 危险操作需要用户手动确认
3. **审计日志**: 所有操作记录在 `~/.opsxcli/opsxcli.db`
4. **权限隔离**: 使用当前用户权限执行,不会越权

## 配置文件

AI 提供商配置存储在 `~/.opsxcli/providers/` 目录:

```bash
# 查看配置目录
ls ~/.opsxcli/providers/

# 输出示例:
# deepseek.json  ollama.json  claude.json
```

### 手动编辑配置

DeepSeek 配置 (`~/.opsxcli/providers/deepseek.json`):
```json
{
  "type": "deepseek",
  "base_url": "https://api.deepseek.com/v1",
  "api_key": "YOUR_API_KEY",
  "model": "deepseek-chat",
  "temperature": 0.7,
  "max_tokens": 4096
}
```

Ollama 配置 (`~/.opsxcli/providers/ollama.json`):
```json
{
  "type": "ollama",
  "base_url": "http://localhost:11434",
  "api_key": "",
  "model": "qwen2.5:latest",
  "temperature": 0.7,
  "max_tokens": 4096
}
```

## 常见问题

### Q1: 如何切换 AI 提供商?

使用 `-p` 参数:
```bash
opsxcli -p deepseek "任务"
opsxcli -p ollama "任务"
```

### Q2: 配置文件在哪里?

```bash
ls ~/.opsxcli/providers/
```

### Q3: 如何查看审计日志?

```bash
sqlite3 ~/.opsxcli/opsxcli.db "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 10;"
```

### Q4: DeepSeek API Key 在哪里获取?

访问 https://platform.deepseek.com/ 注册并在控制台获取。

### Q5: Ollama 如何使用?

```bash
# 1. 安装 Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 2. 下载模型
ollama pull qwen2.5:latest

# 3. 使用
opsxcli -p ollama "任务"
```

### Q6: 为什么有些操作需要确认?

出于安全考虑,危险操作（如删除文件、格式化磁盘）需要手动确认,防止误操作。

### Q7: 使用 AI 会产生费用吗?

- **Ollama**: 完全免费,本地运行
- **DeepSeek**: 按 token 计费,约 ¥1/百万tokens（非常便宜）
- **Claude**: 按 token 计费,费用较高

### Q8: 网络连接失败怎么办?

- 检查网络连接
- 确认 API endpoint 可访问
- 使用 Ollama（本地运行,无需网络）

## 进阶用法

### 指定查询内容

```bash
# 方式1: 直接传参
opsxcli "查看系统信息"

# 方式2: 使用 -q 参数
opsxcli -q "查看系统信息"

# 方式3: 组合使用
opsxcli -p deepseek -q "查看系统信息"
```

### 自定义安全策略

```bash
# 生产环境（严格模式）
opsxcli -s strict "执行任务"

# 开发环境（宽松模式）
opsxcli -s permissive "执行任务"
```

### 批量任务

```bash
# 使用交互模式执行多个相关任务
opsxcli -i
```

## agent 子命令

如果你更喜欢显式的子命令方式,也可以使用 `agent` 子命令:

```bash
# 使用 agent 子命令（功能完全相同）
opsxcli agent "查看系统信息"
opsxcli agent -p deepseek "分析负载"

# 直接调用（推荐,更简洁）
opsxcli "查看系统信息"
opsxcli -p deepseek "分析负载"
```

两种方式功能完全相同,直接调用更简洁,推荐日常使用。

## 相关文档

- [Agent 详细文档](AGENT.md) - AI Agent 架构和实现细节
- [Session 管理](session.md) - 会话历史管理
- [工具注册](../internal/tools/README.md) - 自定义工具开发

## 提示和技巧

1. **描述要具体**: "查看磁盘使用" → "查看根目录磁盘使用情况"
2. **善用交互模式**: 复杂问题可以多轮对话
3. **指定提供商**: 根据场景选择合适的 AI（免费/付费/功能）
4. **注意安全**: 生产环境使用 `strict` 模式
5. **查看日志**: 审计日志可以追踪所有操作

## 更新日志

- **v1.1.0** (2024-12): 添加 `ai` 快捷命令
- **v1.0.0** (2024-12): 首次发布 Agent 功能

## 贡献

欢迎提交 Issue 和 Pull Request!

## 许可证

MIT License
