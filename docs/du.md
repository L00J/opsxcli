# du

opsxcli du — 显示目录或文件的空间使用量

## 用法

`opsxcli du [args...]`

## 说明

估算并显示指定目录或文件的磁盘空间使用量。可用于快速定位占用磁盘空间较大的目录。所有参数直接转发给系统 `du` 命令。

## 选项

此命令无自定义选项，以下为系统 `du` 常用参数：

| 参数 | 说明 |
|------|------|
| `-h` | 以人类可读格式显示 |
| `-s` | 仅显示总计（汇总） |
| `-S` | 不包含子目录的大小 |
| `--max-depth=N` | 显示 N 层子目录的汇总 |
| `-c` | 额外显示总计 |
| `--exclude=PATTERN` | 排除匹配模式的文件 |
| `-a` | 显示所有文件而不仅是目录 |

## 示例

查看目录总大小：

```bash
opsxcli du -sh /path
```

查看一级子目录大小：

```bash
opsxcli du -h --max-depth=1 /path
```

排除特定目录后查看大小：

```bash
opsxcli du -sh --exclude=node_modules /project
```

查看多个目录大小并显示总计：

```bash
opsxcli du -sch /dir1 /dir2
```
