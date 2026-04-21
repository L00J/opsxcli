# chown

opsxcli chown — 修改文件或目录的所有者

## 用法

`opsxcli chown [flags] <owner[:group]> <file> [file...]`

## 说明

修改指定文件或目录的所有者和所属组。所有者格式为数字 UID 或 `UID:GID`。支持同时修改多个文件。

## 选项

| 标志 | 说明 |
|------|------|
| `-R, --recursive` | 递归修改目录及其子内容的所有者 |

## 示例

修改文件所有者为 UID 1000：

```bash
opsxcli chown 1000 file.txt
```

修改文件所有者和组：

```bash
opsxcli chown 1000:1000 data.csv
```

递归修改目录所有者为 root：

```bash
opsxcli chown -R 0:0 /path/to/dir
```
