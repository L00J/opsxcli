# Cp 工具

文件复制工具。

## 使用

```bash
# 复制文件
opsxcli cp source.txt dest.txt

# 复制并重命名
opsxcli cp source.txt /path/to/newfile.txt

# 复制目录
opsxcli cp -r sourcedir/ destdir/

# 保留文件属性
opsxcli cp -p source.txt dest.txt

# 强制覆盖
opsxcli cp -f source.txt dest.txt
```

## 参数

- `-r, --recursive`: 递归复制目录
- `-p, --preserve`: 保留文件属性
- `-f, --force`: 强制覆盖
- `-v, --verbose`: 显示详细输出
