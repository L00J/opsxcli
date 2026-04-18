# Grep 工具

文本搜索工具。

## 使用

```bash
# 基本搜索
opsxcli grep "pattern" file.txt

# 忽略大小写
opsxcli grep -i "pattern" file.txt

# 显示行号
opsxcli grep -n "pattern" file.txt

# 递归搜索目录
opsxcli grep -r "pattern" /path/to/dir

# 显示匹配行的上下文
opsxcli grep -C 2 "pattern" file.txt

# 统计匹配行数
opsxcli grep -c "pattern" file.txt
```

## 参数

- `-i, --ignore-case`: 忽略大小写
- `-n, --line-number`: 显示行号
- `-r, --recursive`: 递归搜索
- `-C, --context`: 显示匹配行的上下文行数
- `-c, --count`: 只显示匹配行数
- `-v, --invert`: 反向匹配（不包含）
