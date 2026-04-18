# Mv 工具

文件移动/重命名工具。

## 使用

```bash
# 移动文件
opsxcli mv source.txt /path/to/dest/

# 重命名文件
opsxcli mv oldname.txt newname.txt

# 移动目录
opsxcli mv sourcedir/ destdir/

# 强制覆盖
opsxcli mv -f source.txt dest.txt
```

## 参数

- `-f, --force`: 强制覆盖
- `-v, --verbose`: 显示详细输出
