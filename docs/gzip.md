# gzip

opsxcli gzip — 压缩或解压文件

## 用法

`opsxcli gzip [args...]`

## 说明

使用 gzip 算法压缩或解压文件。压缩后原文件会被替换为 `.gz` 文件。所有参数直接转发给系统 `gzip` 命令。

## 选项

此命令无自定义选项，以下为系统 `gzip` 常用参数：

| 参数 | 说明 |
|------|------|
| `-d` | 解压文件 |
| `-k` | 保留原文件 |
| `-l` | 列出压缩文件信息 |
| `-r` | 递归压缩目录中的文件 |
| `-v` | 显示压缩/解压过程 |
| `-N` | 指定压缩级别（1-9，默认 6） |
| `-t` | 测试压缩文件完整性 |

## 示例

压缩文件：

```bash
opsxcli gzip file.log
```

解压文件：

```bash
opsxcli gzip -d file.log.gz
```

压缩并保留原文件：

```bash
opsxcli gzip -k file.log
```

使用最高压缩级别：

```bash
opsxcli gzip -9 file.log
```
