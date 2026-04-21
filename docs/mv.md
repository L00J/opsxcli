# mv

opsxcli mv — 移动或重命名文件

## 用法

`opsxcli mv <源路径> <目标路径>`

## 说明

将文件或目录从源路径移动到目标路径，也可用于重命名。此命令无任何选项，仅接受两个参数。

## 选项

无。

## 示例

```bash
# 移动文件
opsxcli mv source.txt /path/to/dest/

# 重命名文件
opsxcli mv oldname.txt newname.txt

# 移动目录
opsxcli mv sourcedir/ destdir/
```
