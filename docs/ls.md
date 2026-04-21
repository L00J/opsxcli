# ls

opsxcli ls — 列出目录内容

## 用法

`opsxcli ls [flags] [目录...]`

## 说明

列出指定目录中的文件和子目录。默认列出当前目录。支持长格式输出、递归列出、排序等选项。

## 选项

- `-a, --all`：显示隐藏文件（以 `.` 开头的文件）
- `-l, --long`：长格式输出，显示文件权限、大小、修改时间等详细信息
- `--human-readable`：以人类可读格式显示文件大小（如 1K、2M）。注意：无短选项 `-h`
- `-R, --recursive`：递归列出子目录内容
- `-t, --time`：按修改时间排序
- `-r, --reverse`：反向排序（与 `-t` 配合使用效果最佳）

## 示例

```bash
# 列出当前目录
opsxcli ls

# 列出指定目录
opsxcli ls /var/log

# 显示隐藏文件
opsxcli ls -a

# 长格式并以人类可读大小显示
opsxcli ls -l --human-readable

# 递归列出所有文件
opsxcli ls -R /etc

# 按时间倒序排列
opsxcli ls -t -r
```
