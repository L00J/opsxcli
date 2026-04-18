# Ls 工具

目录内容列表工具。

## 使用

```bash
# 列出当前目录
opsxcli ls

# 列出指定目录
opsxcli ls /var/log

# 显示隐藏文件
opsxcli ls -a

# 详细列表模式
opsxcli ls -l

# 显示文件大小
opsxcli ls -lh

# 递归列出子目录
opsxcli ls -R
```

## 参数

- `-a, --all`: 显示隐藏文件（以 . 开头）
- `-l`: 详细列表模式
- `-h, --human`: 以人类可读格式显示大小
- `-R, --recursive`: 递归列出子目录
