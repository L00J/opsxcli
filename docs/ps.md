# ps

opsxcli ps — 查看进程信息（转发到系统命令）

## 用法

`opsxcli ps [参数...]`

## 说明

此命令直接转发到系统的 `ps` 命令执行，无自定义选项。所有参数将原样传递给系统 `ps`。

## 选项

无自定义选项。可使用系统 `ps` 命令支持的任何参数。

## 示例

```bash
# 显示当前终端进程
opsxcli ps

# 显示所有进程（完整格式）
opsxcli ps -ef

# 显示进程树
opsxcli ps -ejH

# 查看特定用户进程
opsxcli ps -u root
```
