# Top 工具

实时进程监控工具。

## 使用

```bash
# 启动实时监控
opsxcli top

# 指定刷新间隔
opsxcli top -d 5

# 显示特定用户进程
opsxcli top -u username

# 显示特定进程
opsxcli top -p 1234
```

## 操作

- `q`: 退出
- `P`: 按 CPU 排序
- `M`: 按内存排序
- `T`: 按时间排序
- `k`: 杀死进程
- `r`: 调整优先级

## 参数

- `-d, --delay`: 刷新间隔（秒）
- `-u, --user`: 显示指定用户进程
- `-p, --pid`: 显示指定 PID
