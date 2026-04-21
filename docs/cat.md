# cat

opsxcli cat — 查看文件内容

## 用法

`opsxcli cat [flags] [文件...]`

## 说明

将一个或多个文件的内容输出到标准输出。支持显示行号。可同时查看多个文件。

## 选项

- `-n, --number`：显示行号

## 示例

```bash
# 查看文件内容
opsxcli cat file.txt

# 显示行号
opsxcli cat -n file.txt

# 合并查看多个文件
opsxcli cat file1.txt file2.txt

# 带行号合并查看
opsxcli cat -n file1.txt file2.txt

# 配合管道使用
opsxcli cat -n config.yaml | head -20
```
