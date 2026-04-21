# head

opsxcli head — 显示文件开头内容

## 用法

`opsxcli head [flags] [file...]`

## 说明

输出文件的前 N 行内容。如未指定文件，则从标准输入读取。支持同时读取多个文件。

## 选项

| 标志 | 说明 |
|------|------|
| `-n, --lines` | 显示的行数（默认 10） |

## 示例

显示文件前 10 行：

```bash
opsxcli head config.yaml
```

显示文件前 20 行：

```bash
opsxcli head -n 20 server.log
```

从标准输入读取前 5 行：

```bash
cat largefile.txt | opsxcli head -n 5
```
