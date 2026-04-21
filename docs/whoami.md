# whoami

opsxcli whoami — 显示当前登录用户名

## 用法

`opsxcli whoami`

## 说明

显示当前有效用户名。等效于 `id -un`，用于快速确认当前执行命令的用户身份。参数直接转发给系统 `whoami` 命令。

## 选项

此命令无自定义选项。

## 示例

显示当前用户名：

```bash
opsxcli whoami
```

结合其他命令使用，例如以当前用户身份创建目录：

```bash
mkdir /home/$(opsxcli whoami)/workspace
```
