# grep

opsxcli grep — 文本模式搜索

## 用法

`opsxcli grep [flags] <模式> <文件...>`

## 说明

在文件中搜索匹配指定模式的文本行。支持忽略大小写、反向匹配、显示行号和统计匹配行数。仅支持文件搜索，不支持递归搜索目录。

## 选项

- `-i, --ignore-case`：忽略大小写
- `-v, --invert-match`：反向匹配，显示不匹配的行
- `-n, --line-number`：显示行号
- `-c, --count`：只显示匹配的行数

## 示例

```bash
# 基本搜索
opsxcli grep "error" app.log

# 忽略大小写搜索
opsxcli grep -i "error" app.log

# 显示行号
opsxcli grep -n "error" app.log

# 统计匹配行数
opsxcli grep -c "error" app.log

# 反向匹配（显示不包含 error 的行）
opsxcli grep -v "debug" app.log
```
