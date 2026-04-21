# rm

opsxcli rm — 删除文件或目录

## 用法

`opsxcli rm [flags] <路径...>`

## 说明

删除指定的文件或目录。删除目录时需要使用 `-r` 选项。使用 `-f` 选项可跳过确认直接删除。

## 选项

- `-r, --recursive`：递归删除目录及其内容
- `-f, --force`：强制删除，不提示确认

## 示例

```bash
# 删除文件
opsxcli rm file.txt

# 删除目录
opsxcli rm -r directory/

# 强制删除文件
opsxcli rm -f file.txt

# 强制递归删除目录
opsxcli rm -rf directory/

# 删除多个文件
opsxcli rm file1.txt file2.txt file3.txt
```

## 警告

此操作不可恢复，请谨慎使用。
