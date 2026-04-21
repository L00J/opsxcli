# chmod

opsxcli chmod — 修改文件或目录权限

## 用法

`opsxcli chmod [flags] <mode> <file> [file...]`

## 说明

修改指定文件或目录的访问权限。权限模式使用八进制数字表示，如 `755`、`644` 等。支持同时修改多个文件。

## 选项

| 标志 | 说明 |
|------|------|
| `-R, --recursive` | 递归修改目录及其子内容的权限 |

## 示例

设置脚本可执行权限：

```bash
opsxcli chmod 755 deploy.sh
```

设置文件只读权限：

```bash
opsxcli chmod 644 config.yaml
```

递归修改目录权限：

```bash
opsxcli chmod -R 644 /path/to/dir
```

递归设置目录可执行：

```bash
opsxcli chmod -R 755 /opt/app/bin
```
