# agent

opsxcli agent — AI 驱动的运维助手

## 用法

`opsxcli agent [query] [flags]`

## 说明

AI 驱动的智能运维助手，使用自然语言描述运维问题，自动调用相关工具分析并解决问题。支持交互式对话、安全模式控制、后台任务管理及会话持久化。

## 选项

| 标志 | 说明 |
|------|------|
| `-p, --provider` | 指定 LLM 提供商 |
| `-s, --safety` | 安全模式（`strict`/`balanced`/`permissive`，默认 `balanced`） |
| `-i, --interactive` | 启用交互模式 |
| `-q, --query` | 查询内容 |
| `-d, --debug` | 调试模式，输出详细日志 |
| `-y, --yes` | 自动批准所有操作（跳过确认） |
| `-b, --background` | 启用后台任务管理 |
| `--resume` | 恢复指定会话 |
| `--list-sessions` | 列出历史会话 |
| `--export` | 导出会话为 Markdown |

## 示例

```bash
# 单次查询
opsxcli agent "检查服务器负载并分析瓶颈"

# 交互模式
opsxcli agent -i

# 指定提供商和调试模式
opsxcli agent -p openai -d "清理磁盘空间"

# 恢复历史会话
opsxcli agent --resume abc123

# 列出所有历史会话
opsxcli agent --list-sessions
```
