# Cat 工具

查看文件内容。

## 使用

```bash
# 查看文件内容
opsxcli cat file.txt

# 显示行号
opsxcli cat -n file.txt

# 显示不可见字符
opsxcli cat -A file.txt

# 合并多个文件
opsxcli cat file1.txt file2.txt
```

## 参数

- `-n, --number`: 显示行号
- `-A, --show-all`: 显示所有字符（包括不可见字符）
