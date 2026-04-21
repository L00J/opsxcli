# date

opsxcli date — 显示或设置系统日期时间

## 用法

`opsxcli date [args...]`

## 说明

转发命令，将所有参数直接传递给系统 `date` 命令。用于显示当前日期时间、格式化输出或设置系统时间。

此命令无自定义选项，完全兼容系统 `date` 的所有参数和格式化字符串。

## 选项

本命令为转发命令，无自定义标志。所有参数直接传递给系统 `date`。

## 示例

```bash
# 显示当前日期时间
opsxcli date

# 以 ISO 8601 格式输出
opsxcli date --iso-8601

# 自定义格式输出
opsxcli date +"%Y-%m-%d %H:%M:%S"

# 显示 UTC 时间
opsxcli date -u
```
