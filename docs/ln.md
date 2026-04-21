# ln

opsxcli ln — 创建文件链接

## 用法

`opsxcli ln [flags] <target> <link>`

## 说明

在目标文件和链接路径之间创建链接。默认创建硬链接，使用 `-s` 选项创建符号链接（软链接）。

## 选项

| 标志 | 说明 |
|------|------|
| `-s, --symbolic` | 创建符号链接而非硬链接 |

## 示例

创建硬链接：

```bash
opsxcli ln file.txt hardlink.txt
```

创建符号链接：

```bash
opsxcli ln -s /path/to/file symlink
```

为可执行文件创建软链接：

```bash
opsxcli ln -s /opt/app/bin/run /usr/local/bin/myapp
```
