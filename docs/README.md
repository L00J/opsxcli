# opsxcli 工具文档

## 快速导航

### 文件操作
- [ls](ls.md) - 列出目录内容
- [cat](cat.md) - 查看文件内容
- [grep](grep.md) - 文本搜索
- [cp](cp.md) - 复制文件/目录
- [mv](mv.md) - 移动/重命名
- [rm](rm.md) - 删除文件
- [mkdir](mkdir.md) - 创建目录

### 数据库
- [mysql](mysql.md) - MySQL 数据库操作
- [psql](psql.md) - PostgreSQL 数据库操作
- [redis](redis.md) - Redis 操作

### 系统工具
- [ps](ps.md) - 进程查看
- [top](top.md) - 实时进程监控
- [dd](dd.md) - 磁盘数据复制

### 网络工具
- [ssh](ssh.md) - SSH 连接和文件传输
- [ping](ping.md) - 网络连通性测试
- [traceroute](traceroute.md) - 路由追踪
- [telnet](telnet.md) - Telnet 客户端
- [nc](nc.md) - 网络瑞士军刀
- [ss](ss.md) - 网络连接状态
- [netstat](netstat.md) - 网络统计
- [nmap](nmap.md) - 端口扫描

### 网络配置
- [ifconfig](ifconfig.md) - 网络接口配置
- [route](route.md) - 路由表管理
- [ip](ip.md) - IP 地址管理

### HTTP 工具
- [curl](curl.md) - HTTP 请求
- [wget](wget.md) - 文件下载
- [request](request.md) - 高级 HTTP 请求
- [websearch](websearch.md) - 网络搜索

### 服务
- [server](server.md) - HTTP 服务器

### Docker
- [docker-pull](docker-pull-accelerator.md) - Docker 镜像加速

### Kubernetes
- [kubectl](kubectl.md) - kubectl 命令代理
- [consul](consul.md) - K8s 服务注册到 Consul
- [kubernetes](kubernetes.md) - K8s 资源管理

### 监控
- [sys](sys.md) - 系统监控
- [net](net.md) - 网络监控

### 管理
- [install](install.md) - 系统包管理器
- [upgrade](upgrade.md) - 升级 opsxcli

## 架构文档
- [ARCHITECTURE](ARCHITECTURE.md) - 项目架构说明
- [PLUGIN](PLUGIN_ARCHITECTURE.md) - 插件架构
- [BUSYBOX](BUSYBOX_INTEGRATION.md) - Busybox 集成

## 使用示例

```bash
# 查看帮助
opsxcli --help
opsxcli <command> --help

# 系统监控
opsxcli sys

# 网络监控
opsxcli net

# SSH 连接
opsxcli ssh root@host

# MySQL 操作
opsxcli mysql -u root -p "password" -h localhost

# Redis 操作
opsxcli redis -h 127.0.0.1

# Docker 镜像加速
opsxcli docker pull nginx:latest
```
