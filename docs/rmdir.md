# rmdir

opsxcli rmdir — 删除空目录

## 用法

`opsxcli rmdir [flags] <dir> [dir...]`

## 说明

删除指定的空目录。可同时删除多个目录。若目录非空则操作失败。

## 选项

| 标志 | 说明 |
|------|------|
| `-p, --parents` | 删除目录及其祖先目录（类似 `rmdir -p a/b/c` 依次删除 c、b、a） |

## 示例

删除单个空目录：

```bash
opsxcli rmdir /tmp/emptydir
```

同时删除多个空目录：

```bash
opsxcli rmdir dir1 dir2 dir3
```

删除目录及父目录：

```bash
opsxcli rmdir -p /tmp/a/b/c
```
