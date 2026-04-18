# Netstat 工具

网络连接状态查看工具，显示网络连接信息。

## 使用

```bash
# 显示监听状态的连接
opsxcli netstat -l

# 显示所有连接
opsxcli netstat -a

# 显示TCP连接
opsxcli netstat -t

# 显示UDP连接
opsxcli netstat -u

# 显示PID和程序名
opsxcli netstat -p

# 组合使用
opsxcli netstat -pnutl
```

## 参数

- `-l, --listen`: 只显示监听状态的连接
- `-a, --all`: 显示所有连接
- `-t, --tcp`: 显示TCP连接
- `-u, --udp`: 显示UDP连接
- `-n, --numeric`: 以数字形式显示地址和端口
- `-p, --programs`: 显示PID和程序名
