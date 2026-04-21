# tar

opsxcli tar — 归档工具

## 用法

`opsxcli tar [args...]`

## 说明

创建、提取或管理 tar 归档文件，支持配合 gzip、bzip2、xz 等压缩格式使用。所有参数直接转发给系统 `tar` 命令。

## 选项

此命令无自定义选项，以下为系统 `tar` 常用参数：

| 参数 | 说明 |
|------|------|
| `-c` | 创建新归档 |
| `-x` | 提取归档 |
| `-t` | 列出归档内容 |
| `-z` | 使用 gzip 压缩/解压 |
| `-j` | 使用 bzip2 压缩/解压 |
| `-J` | 使用 xz 压缩/解压 |
| `-v` | 显示处理过程 |
| `-f FILE` | 指定归档文件名 |

## 示例

创建 gzip 压缩归档：

```bash
opsxcli tar czf backup.tar.gz /path/to/dir
```

解压 gzip 归档：

```bash
opsxcli tar xzf backup.tar.gz
```

列出归档内容：

```bash
opsxcli tar tzf backup.tar.gz
```

解压到指定目录：

```bash
opsxcli tar xzf backup.tar.gz -C /target/dir
```
