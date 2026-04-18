# Rm 工具

文件删除工具。

## 使用

```bash
# 删除文件
opsxcli rm file.txt

# 删除目录
opsxcli rm -r directory/

# 强制删除
opsxcli rm -f file.txt

# 删除前确认
opsxcli rm -i file.txt
```

## 参数

- `-r, --recursive`: 递归删除目录
- `-f, --force`: 强制删除
- `-i, --interactive`: 删除前确认
- `-v, --verbose`: 显示详细输出

## 警告

此操作不可恢复，请谨慎使用。
