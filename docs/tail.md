# tail

opsxcli tail — 显示文件末尾内容

## 用法

`opsxcli tail [flags] [file...]`

## 说明

输出文件的最后 N 行内容。如未指定文件，则从标准输入读取。常用于查看日志文件的最新记录。

## 选项

| 标志 | 说明 |
|------|------|
| `-n, --lines` | 显示的行数（默认 10） |

## 示例

查看日志最后 10 行：

```bash
opsxcli tail app.log
```

查看日志最后 50 行：

```bash
opsxcli tail -n 50 error.log
```

从管道读取末尾内容：

```bash
cat access.log | opsxcli tail -n 20
```
