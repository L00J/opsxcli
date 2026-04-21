# sleep

opsxcli sleep — 延迟指定时间

## 用法

`opsxcli sleep <duration>`

## 说明

转发命令，将参数直接传递给系统 `sleep` 命令。用于在脚本中暂停执行指定的时间。

支持数字后缀：`s`（秒，默认）、`m`（分）、`h`（时）、`d`（天）。

## 选项

本命令为转发命令，无自定义标志。所有参数直接传递给系统 `sleep`。

## 示例

```bash
# 延迟 5 秒
opsxcli sleep 5

# 延迟 2 分钟
opsxcli sleep 2m

# 延迟 1 小时
opsxcli sleep 1h
```
