# unzip

opsxcli unzip — 解压 ZIP 文件

## 用法

`opsxcli unzip [args...]`

## 说明

解压 ZIP 格式的压缩归档文件，可指定解压目标目录或仅查看内容。所有参数直接转发给系统 `unzip` 命令。

## 选项

此命令无自定义选项，以下为系统 `unzip` 常用参数：

| 参数 | 说明 |
|------|------|
| `-d DIR` | 解压到指定目录 |
| `-l` | 列出压缩包内容 |
| `-o` | 覆盖已存在的文件 |
| `-q` | 静默模式 |
| `-v` | 显示详细信息 |
| `-x PATTERN` | 排除匹配的文件 |
| `-t` | 测试压缩文件完整性 |

## 示例

解压到当前目录：

```bash
opsxcli unzip archive.zip
```

解压到指定目录：

```bash
opsxcli unzip archive.zip -d /target
```

列出压缩包内容：

```bash
opsxcli unzip -l archive.zip
```

解压时覆盖已有文件：

```bash
opsxcli unzip -o archive.zip
```
