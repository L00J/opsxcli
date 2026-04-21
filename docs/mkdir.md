# mkdir

opsxcli mkdir — 创建目录

## 用法

`opsxcli mkdir [flags] <目录名...>`

## 说明

创建指定目录。默认权限为 0755。使用 `-p` 选项可自动创建缺失的父目录。

## 选项

- `-p, --parents`：自动创建父目录，若目录已存在不报错

## 示例

```bash
# 创建单个目录
opsxcli mkdir newdir

# 创建多级目录
opsxcli mkdir -p path/to/nested/dir

# 同时创建多个目录
opsxcli mkdir dir1 dir2 dir3

# 创建嵌套目录结构
opsxcli mkdir -p /opt/app/{config,data,logs}
```
