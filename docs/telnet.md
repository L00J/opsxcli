# Telnet 工具

Telnet客户端工具，支持连接和监听模式。

## 连接模式

```bash
# 连接到远程主机
opsxcli telnet 192.168.1.100 23
opsxcli telnet example.com 80 -t 5s -v
```

## 监听模式

```bash
# 监听端口作为Telnet服务器
opsxcli telnet -l -p 2323
opsxcli telnet --listen --port 8080 -v
```

## 参数

- `-l, --listen`: 监听模式（服务器模式）
- `-p, --port`: 端口号（默认: 23）
- `-v, --verbose`: 详细输出
- `-t, --timeout`: 连接超时时间（默认: 10s）
