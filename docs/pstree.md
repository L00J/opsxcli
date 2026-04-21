# pstree

opsxcli pstree — 以树形结构显示进程关系

## 用法

`opsxcli pstree [args...]`

## 说明

以树状图形式显示系统中的进程及其父子关系，便于直观了解进程层级结构。所有参数直接转发给系统 `pstree` 命令。

## 选项

此命令无自定义选项，以下为系统 `pstree` 常用参数：

| 参数 | 说明 |
|------|------|
| `-p` | 显示进程 PID |
| `-u` | 显示进程所属用户 |
| `-h` | 高亮当前进程及其祖先 |
| `-l` | 不截断长行 |
| `-s` | 显示父进程 |
| `-n` | 按 PID 排序 |

## 示例

显示完整进程树：

```bash
opsxcli pstree
```

显示包含 PID 的进程树：

```bash
opsxcli pstree -p
```

查看指定进程的子进程树：

```bash
opsxcli pstree 1234
```

高亮当前进程并显示 PID：

```bash
opsxcli pstree -hp
```
