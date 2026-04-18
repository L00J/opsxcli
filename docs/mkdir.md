# Mkdir 工具

创建目录工具。

## 使用

```bash
# 创建单个目录
opsxcli mkdir newdir

# 创建多级目录
opsxcli mkdir -p path/to/nested/dir

# 创建目录并设置权限
opsxcli mkdir -m 755 newdir
```

## 参数

- `-p, --parents`: 创建多级目录
- `-m, --mode`: 设置目录权限
