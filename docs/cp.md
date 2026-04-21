# cp

opsxcli cp — 复制文件或目录

## 用法

`opsxcli cp [flags] <源路径> <目标路径>`

## 说明

复制文件或目录到指定目标路径。复制目录时需要使用 `-r` 选项。

## 选项

- `-r, --recursive`：递归复制目录及其内容

## 示例

```bash
# 复制文件
opsxcli cp source.txt dest.txt

# 复制并重命名
opsxcli cp source.txt /path/to/newfile.txt

# 复制目录
opsxcli cp -r sourcedir/ destdir/

# 复制文件到指定目录
opsxcli cp config.yaml /etc/app/
```
