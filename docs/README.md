# opsxcli 工具文档

## 📚 文档索引

### 核心文档
- [项目总结](SUMMARY.md) - opsxcli 完整项目总结
- [性能优化](OPTIMIZATION.md) - 体积优化、性能优化总结
- [网络工具替代](NETWORK_TOOLS_REPLACEMENT.md) - 完整替代 iproute/net-tools
- [Busybox 集成](BUSYBOX_INTEGRATION.md) - Busybox 命令集成规划

## 数据库工具

- [MySQL](mysql.md) - MySQL数据库操作工具（交互式 shell）
- [PostgreSQL](psql.md) - PostgreSQL数据库操作工具
- [Redis](redis.md) - Redis操作工具（支持单机和集群）

## 网络工具

- [SSH](ssh.md) - SSH连接、命令执行、文件传输、端口转发
- [Telnet](telnet.md) - Telnet客户端工具
- [NC](nc.md) - 网络连接工具（端口监听、内网反弹）
- [Ping](ping.md) - 网络连通性测试
- [Traceroute](traceroute.md) - 路由追踪
- [Netstat](netstat.md) - 网络连接状态查看
- [SS](ss.md) - 网络连接状态查看（高性能，替代 netstat）
- [Nmap](nmap.md) - 网络端口扫描

## 监控工具

- [Sys](sys.md) - 系统监控可视化（类似 htop，2秒实时刷新）
- [Net](net.md) - 网络流量监控（类似 iftop，2秒实时刷新）

## 服务端工具

- [Server](server.md) - HTTP/WebSocket/gRPC 服务端

## 其他工具

- [Request](request.md) - 高级HTTP请求工具
- [Wget](wget.md) - 文件下载工具（支持断点续传）
- [Install](install.md) - 自动包管理器安装工具

## 快速开始

```bash
# 查看帮助
opsxcli --help

# 系统监控（类似 htop）
opsxcli sys

# 网络监控（类似 iftop）
opsxcli net

# 网络连接状态
opsxcli ss -tunap

# SSH 连接
opsxcli ssh user@host

# MySQL 连接
opsxcli mysql -u root -p "password" -h localhost
```
