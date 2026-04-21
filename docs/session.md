# session

opsxcli session — 管理 AI Agent 对话会话

## 用法

`opsxcli session <subcommand> [flags]`

## 说明

管理 AI Agent 的对话会话，支持列出、恢复、导出、删除和重命名会话。会话记录持久化存储，可随时恢复历史对话上下文。

## 子命令

### `list`

列出所有历史会话。

| 标志 | 说明 |
|------|------|
| `-n, --limit` | 显示数量（默认 `20`） |

### `resume <id>`

恢复指定会话，继续对话。

| 标志 | 说明 |
|------|------|
| `-p, --provider` | 覆盖 LLM 提供商 |

### `export <id>`

导出会话内容。

| 标志 | 说明 |
|------|------|
| `-f, --format` | 导出格式（`markdown`/`json`，默认 `markdown`） |
| `-o, --output` | 输出文件路径（`-` 表示 stdout） |

### `delete <id>`

删除指定会话。

### `rename <id> <title>`

重命名指定会话。

## 示例

```bash
# 列出最近 20 条会话
opsxcli session list

# 恢复指定会话
opsxcli session resume abc123

# 导出会话为 JSON 文件
opsxcli session export abc123 -f json -o session.json

# 删除会话
opsxcli session delete abc123

# 重命名会话
opsxcli session rename abc123 "磁盘排查记录"
```
