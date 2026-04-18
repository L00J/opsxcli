# Ps 工具

进程查看工具。

## 使用

```bash
# 显示所有进程
opsxcli ps

# 显示所有进程（详细）
opsxcli ps -ef

# 显示进程树
opsxcli ps -ejH

# 按 CPU 使用排序
opsxcli ps --sort=-cpu

# 按内存使用排序
opsxcli ps --sort=-mem

# 查看特定用户进程
opsxcli ps -u username
```

## 参数

- `-e`: 显示所有进程
- `-f`: 显示完整格式
- `-u`: 按用户显示
- `-j`: 显示作业格式
- `-H`: 显示进程树
- `--sort`: 排序（-cpu, -mem）
