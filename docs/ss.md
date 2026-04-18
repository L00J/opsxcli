# Ss 工具

网络连接状态查看工具（高性能替代 netstat）。

## 使用

```bash
# 显示所有连接
opsxcli ss

# 显示监听端口
opsxcli ss -l

# 显示 TCP 连接
opsxcli ss -t

# 显示 UDP 连接
opsxcli ss -u

# 显示进程信息
opsxcli ss -p

# 显示概要统计
opsxcli ss -s

# 显示计时器信息
opsxcli ss -o

# 目标地址 TOP
opsxcli ss -ant --top 10
```

## 参数

- `-l, --listen`: 显示监听端口
- `-t, --tcp`: TCP 连接
- `-u, --udp`: UDP 连接
- `-p, --processes`: 显示进程
- `-n, --numeric`: 数字格式
- `-a, --all`: 所有连接
- `-s, --summary`: 概要统计
- `-o, --options`: 计时器信息
- `--top N`: 显示 TOP N 连接

## 示例

```bash
# TIME_WAIT 状态分析
opsxcli ss -tan --timewait

# TCP 状态统计
opsxcli ss -ant --stats
```
