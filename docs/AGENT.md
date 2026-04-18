# opsxcli Agent - AI运维助手

## 功能概述

opsxcli Agent 是一个基于大语言模型的AI运维助手，可以通过自然语言理解运维问题，自动调用相关工具来解决问题。

### 主要特性

- 🤖 **智能问题解决**：使用自然语言描述问题，Agent自动分析并调用工具
- 🔧 **工具集成**：集成40+运维工具（文件操作、系统监控、网络诊断、Kubernetes等）
- 🛡️ **安全可控**：多级风险分类，危险操作需用户确认
- 📝 **操作审计**：所有操作自动记录审计日志
- 🎯 **ReAct循环**：智能推理和行动循环，自动多步骤问题解决

## 快速开始

### 1. 首次使用

首次运行会自动创建配置文件：

```bash
opsxcli agent "测试"
```

输出：
```
🔧 首次使用，创建默认配置...
✓ 配置文件已创建在: /root/.opsxcli/providers/
⚠️  请先编辑配置文件，填入API密钥后再使用
```

### 2. 配置LLM提供商

编辑配置文件（三选一）：

#### 使用 Ollama（本地模型，推荐入门）

```bash
vim ~/.opsxcli/providers/ollama.json
```

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

先安装Ollama:
```bash
curl -fsSL https://ollama.com/install.sh | sh
ollama pull qwen2.5:latest
```

#### 使用 DeepSeek（性价比高）

```bash
vim ~/.opsxcli/providers/deepseek.json
```

```json
{
  "type": "deepseek",
  "base_url": "https://api.deepseek.com/v1",
  "api_key": "YOUR_DEEPSEEK_API_KEY",
  "model": "deepseek-chat",
  "temperature": 0.7,
  "max_tokens": 4096
}
```

获取API Key: https://platform.deepseek.com/

#### 使用 Claude（功能最强）

```bash
vim ~/.opsxcli/providers/claude.json
```

```json
{
  "type": "claude",
  "base_url": "https://api.anthropic.com/v1",
  "api_key": "YOUR_CLAUDE_API_KEY",
  "model": "claude-3-5-sonnet-20241022",
  "temperature": 0.7,
  "max_tokens": 4096
}
```

获取API Key: https://console.anthropic.com/

### 3. 使用Agent

#### 单次查询模式

```bash
# 查看磁盘使用
opsxcli agent "查看根目录磁盘使用情况"

# 查找进程
opsxcli agent "查找所有监听80端口的进程"

# Kubernetes诊断
opsxcli agent "查看default命名空间下所有处于Pending状态的pods"

# 系统诊断
opsxcli agent "分析系统内存使用情况，找出占用最多的进程"
```

#### 交互模式

```bash
opsxcli agent -i
```

交互式对话：
```
🤖 opsxcli Agent - AI运维助手
输入 'exit' 或 'quit' 退出

👤 您: 查看系统负载
=== 迭代 1/10 ===
🔧 工具调用数量: 1
  → 工具: top
  → 参数: {}
  → 风险等级: safe
  ✓ 安全检查通过，开始执行...
  ✓ 执行成功

🤖 Agent:
系统负载适中，CPU使用率45%，内存使用率62%...

👤 您: exit
👋 再见！
```

#### 指定LLM提供商

```bash
# 使用DeepSeek
opsxcli agent -p deepseek "查看磁盘使用"

# 使用Claude
opsxcli agent -p claude "分析网络连接"

# 使用Ollama
opsxcli agent -p ollama "查看系统信息"
```

#### 调整安全级别

```bash
# 严格模式：中风险及以上需要确认
opsxcli agent -s strict "删除临时文件"

# 平衡模式：高风险及以上需要确认（默认）
opsxcli agent -s balanced "查看日志"

# 宽松模式：只有危险操作需要确认
opsxcli agent -s permissive "重启服务"
```

## 工具列表

### 文件操作（安全 🟢）
- `cat` - 查看文件内容
- `grep` - 搜索文件内容
- `ls` - 列出目录
- `head` - 查看文件开头
- `tail` - 查看文件末尾
- `tree` - 树形显示目录

### 系统信息（安全 🟢）
- `ps` - 查看进程
- `top` - 实时进程监控
- `free` - 内存使用
- `df` - 磁盘空间
- `du` - 目录大小
- `uname` - 系统信息

### 网络工具（安全 🟢 / 中风险 🟠）
- `ping` - 测试连通性
- `ss` - 查看网络连接
- `netstat` - 网络状态
- `ifconfig` - 网络接口
- `ssh` - SSH连接（中风险 🟠）

### Kubernetes（安全 🟢 / 高风险 🔴）
- `kubectl_get` - 获取资源
- `kubectl_describe` - 查看详情
- `kubectl_logs` - 查看日志
- `kubectl_delete` - 删除资源（高风险 🔴）

### 危险操作（危险 💀）
- `rm` - 删除文件（危险 💀）
- `dd` - 磁盘操作（危险 💀）

## 风险等级说明

| 等级 | 标识 | 说明 | 确认要求 |
|------|------|------|----------|
| 安全 | 🟢 | 只读操作 | 无需确认 |
| 低风险 | 🟡 | 轻微修改 | 宽松模式无需确认 |
| 中风险 | 🟠 | 文件操作 | 平衡/严格模式需确认 |
| 高风险 | 🔴 | 系统配置 | 所有模式需确认 |
| 危险 | 💀 | 删除/格式化 | 必须确认 |

## 审计日志

所有操作都会记录审计日志：

```bash
# 查看审计日志数据库
sqlite3 ~/.opsxcli/opsxcli.db "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 10;"
```

日志包含：
- 用户信息
- 工具名称和参数
- 风险等级
- 审批状态
- 执行结果
- 时间戳

## 使用示例

### 示例1：系统诊断

```bash
opsxcli agent "服务器响应慢，帮我排查原因"
```

Agent会自动：
1. 使用`top`查看系统负载
2. 使用`free`检查内存使用
3. 使用`df`检查磁盘空间
4. 使用`ss`检查网络连接
5. 分析结果并给出建议

### 示例2：Kubernetes问题排查

```bash
opsxcli agent "default命名空间有pod一直重启，帮我找出原因"
```

Agent会自动：
1. 使用`kubectl_get`列出所有pods
2. 找出重启次数高的pod
3. 使用`kubectl_logs`查看日志
4. 使用`kubectl_describe`查看事件
5. 分析并给出解决方案

### 示例3：安全文件查看

```bash
opsxcli agent "查看/var/log/messages中最近的错误日志"
```

Agent会自动：
1. 使用`tail`查看最新日志
2. 使用`grep`过滤错误信息
3. 分析错误模式
4. 给出处理建议

## 注意事项

1. **API密钥安全**：配置文件权限为600，仅所有者可读写
2. **危险操作确认**：删除、dd等操作必须手动确认
3. **网络环境**：使用云端API需要网络连接
4. **成本控制**：API调用会产生费用（Ollama除外）
5. **工具权限**：Agent使用当前用户权限执行命令

## 架构设计

```
opsxcli agent
    │
    ├── LLM客户端层 (internal/llm)
    │   ├── OpenAI兼容客户端 (DeepSeek, Ollama)
    │   ├── Claude客户端
    │   └── 配置管理
    │
    ├── 工具注册层 (internal/tools)
    │   ├── 工具接口定义
    │   ├── 命令行工具包装器
    │   └── 风险等级分类
    │
    ├── 安全控制层 (internal/agent/safety)
    │   ├── 风险评估
    │   ├── 用户确认
    │   └── 审计日志
    │
    ├── Agent引擎 (internal/agent/agent)
    │   ├── ReAct循环
    │   ├── 工具调用
    │   └── 上下文管理
    │
    └── CLI界面 (cmd/agent)
        ├── 单次查询
        └── 交互模式
```

## 故障排查

### 问题1：API调用失败

```
错误: LLM调用失败: dial tcp: lookup api.deepseek.com: no such host
```

**解决**：检查网络连接，确认可以访问API

### 问题2：工具执行失败

```
错误: 工具执行异常: executable file not found in $PATH
```

**解决**：确保opsxcli在PATH中，或使用完整路径

### 问题3：权限不足

```
错误: permission denied
```

**解决**：使用sudo或切换到有权限的用户

## 后续计划

- [ ] Web界面（后续独立开发）
- [ ] 更多工具支持（MySQL、Redis等）
- [ ] 流式输出
- [ ] 会话历史管理
- [ ] 工具使用统计

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

与opsxcli主项目相同
