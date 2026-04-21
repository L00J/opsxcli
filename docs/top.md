# top

opsxcli top — 实时进程监控（转发到系统命令）

## 用法

`opsxcli top [参数...]`

## 说明

此命令直接转发到系统的 `top` 命令执行，无自定义选项。禁用了命令行参数解析（DisableFlagParsing），所有参数将原样传递给系统 `top`。

## 选项

无自定义选项。可使用系统 `top` 命令支持的任何参数。

## 示例

```bash
# 启动实时监控
opsxcli top

# 指定刷新间隔
opsxcli top -d 5

# 显示特定用户进程
opsxcli top -u root

# 批处理模式输出
opsxcli top -b -n 1
```
