# tree

opsxcli tree — 树形显示目录结构

## 用法

`opsxcli tree [args...]`

## 说明

以树形结构显示当前或指定目录的文件和子目录。本命令为转发命令，所有参数直接传递给系统 `tree` 命令。

前提条件：系统需已安装 `tree` 命令。

## 选项

无自定义选项，所有参数转发给系统 `tree` 命令。

## 示例

显示当前目录树：

```bash
opsxcli tree
```

显示指定目录树：

```bash
opsxcli tree /path/to/project
```

限制显示深度：

```bash
opsxcli tree -L 2 /opt/app
```

只显示目录：

```bash
opsxcli tree -d /etc
```
