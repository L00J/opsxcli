# ss

opsxcli ss — 网络连接状态查看工具

## 用法

`opsxcli ss [flags]`

## 说明

查看网络连接状态，支持过滤 TCP/UDP 连接、监听端口、进程信息、TCP 状态统计、TIME_WAIT 分析和目标地址 TOP 统计。

## 选项

- `-l, --listen`：只显示监听状态的连接
- `-a, --all`：显示所有连接（包括监听的和非监听的）
- `-t, --tcp`：只显示 TCP 连接
- `-u, --udp`：只显示 UDP 连接
- `-n, --numeric`：以数字格式显示地址和端口（不解析域名）
- `-p, --programs`：显示进程 PID 和程序名
- `-s, --stats`：统计各 TCP 状态的连接数量
- `-w, --timewait`：显示 TIME_WAIT 状态 TOP 10
- `--top N`：显示目标地址 TOP N 连接数

## 示例

```bash
# 显示所有 TCP 连接
opsxcli ss -t

# 显示所有监听端口
opsxcli ss -l

# 显示 TCP 监听端口及进程信息
opsxcli ss -tlp

# TCP 状态统计
opsxcli ss -s

# TIME_WAIT TOP 10 分析
opsxcli ss -tw

# 目标地址 TOP 20
opsxcli ss -tan --top 20
```
