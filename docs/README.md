# opsxcli 文档

## 设计文档（按顺序阅读）

| 编号 | 文档 | 核心内容 |
|------|------|----------|
| 01 | [项目概述与设计哲学](01-项目概述与设计哲学.md) | 项目定位、四大设计哲学、Agent 全景图 |
| 02 | [整体架构设计](02-整体架构设计.md) | cmd/plugins/internal 三层架构详解 |
| 03 | [Agent 核心引擎设计](03-Agent核心引擎设计.md) | ReAct 主循环、自适应检查点、16 轮迭代 |
| 04 | [插件架构设计](04-插件架构设计.md) | 插件分类、注册机制、扩展规范 |
| 05 | [Busybox 集成方案](05-Busybox集成方案.md) | 基础命令 Go 原生实现策略 |
| 06 | [Prompt 构建与记忆注入](06-Prompt构建与记忆注入.md) | 五层 System Prompt + 动态记忆层 |
| 07 | [Evolver 自进化引擎](07-Evolver自进化引擎.md) | 三层记忆、经验学习、自我进化 |
| 08 | [安全系统与工具注册](08-安全系统与工具注册.md) | 风险评估、三级安全模式、Tool 接口扩展 |
| 09 | [会话管理与 TUI 界面](09-会话管理与TUI界面.md) | JSONL 持久化、Bubble Tea 终端 UI |

---

## 命令文档

### 文件操作

| 命令 | 说明 |
|------|------|
| [ls](ls.md) | 列出目录内容 |
| [cat](cat.md) | 显示文件内容 |
| [head](head.md) | 显示文件开头 |
| [tail](tail.md) | 显示文件末尾 |
| [cp](cp.md) | 复制文件/目录 |
| [mv](mv.md) | 移动/重命名文件 |
| [rm](rm.md) | 删除文件/目录 |
| [mkdir](mkdir.md) | 创建目录 |
| [rmdir](rmdir.md) | 删除空目录 |
| [touch](touch.md) | 创建空文件/更新时间戳 |
| [chmod](chmod.md) | 修改文件权限 |
| [chown](chown.md) | 修改文件所有者 |
| [ln](ln.md) | 创建链接 |
| [tree](tree.md) | 树形显示目录结构 |
| [grep](grep.md) | 文本搜索 |

### 系统信息

| 命令 | 说明 |
|------|------|
| [uname](uname.md) | 显示系统信息 |
| [hostname](hostname.md) | 显示/设置主机名 |
| [whoami](whoami.md) | 显示当前用户 |
| [id](id.md) | 显示用户和组信息 |
| [free](free.md) | 显示内存使用情况 |
| [df](df.md) | 显示磁盘空间 |
| [du](du.md) | 显示目录大小 |
| [dd](dd.md) | 磁盘数据复制 |

### 进程管理

| 命令 | 说明 |
|------|------|
| [ps](ps.md) | 查看进程 |
| [top](top.md) | 实时进程监控 |
| [kill](kill.md) | 终止进程 |
| [pstree](pstree.md) | 显示进程树 |

### 网络工具

| 命令 | 说明 |
|------|------|
| [ping](ping.md) | 网络连通性测试 |
| [traceroute](traceroute.md) | 路由追踪 |
| [telnet](telnet.md) | Telnet 客户端 |
| [nc](nc.md) | 网络瑞士军刀 |
| [ss](ss.md) | 网络连接状态 |
| [netstat](netstat.md) | 网络统计 |
| [nmap](nmap.md) | 端口扫描 |

### 网络配置

| 命令 | 说明 |
|------|------|
| [ifconfig](ifconfig.md) | 网络接口信息（只读） |
| [route](route.md) | 路由表信息（只读） |
| [ip](ip.md) | IP 地址信息（只读） |
| [ssl](ssl.md) | SSL 证书检查 |
| [ssh](ssh.md) | SSH 连接和文件传输 |
| [ssh-config](ssh-config.md) | SSH 配置管理 |

### HTTP 工具

| 命令 | 说明 |
|------|------|
| [curl](curl.md) | HTTP 请求 |
| [wget](wget.md) | 文件下载 |
| [requests](requests.md) | 高级 HTTP 请求 |
| [SimpleHTTPServer](SimpleHTTPServer.md) | HTTP 文件服务器 |

### 数据库

| 命令 | 说明 |
|------|------|
| [mysql](mysql.md) | MySQL 操作 |
| [mysqldump](mysqldump.md) | 导出 MySQL 数据库 |
| [mysqlrestore](mysqlrestore.md) | 导入 MySQL 数据库 |
| [psql](psql.md) | PostgreSQL 操作 |
| [pgdump](pgdump.md) | 导出 PostgreSQL 数据库 |
| [pgrestore](pgrestore.md) | 导入 PostgreSQL 数据库 |
| [redis](redis.md) | Redis 操作 |

### 归档压缩

| 命令 | 说明 |
|------|------|
| [tar](tar.md) | 归档工具 |
| [gzip](gzip.md) | 压缩文件 |
| [unzip](unzip.md) | 解压 ZIP 文件 |

### 时间日期

| 命令 | 说明 |
|------|------|
| [date](date.md) | 显示/设置日期时间 |
| [sleep](sleep.md) | 延迟指定时间 |

### Docker

| 命令 | 说明 |
|------|------|
| [docker](docker.md) | Docker 镜像加速 |

### Kubernetes

| 命令 | 说明 |
|------|------|
| [kubectl](kubectl.md) | kubectl 命令代理 |
| [kubernetes](kubernetes.md) | K8s 资源管理 |
| [consul](consul.md) | K8s 服务注册到 Consul |

### 监控

| 命令 | 说明 |
|------|------|
| [sys](sys.md) | 系统监控 |
| [net](net.md) | 网络监控 |
| [logs](logs.md) | 日志分析 |

### 压测与通知

| 命令 | 说明 |
|------|------|
| [bench](bench.md) | HTTP 压力测试 |
| [notify](notify.md) | 告警通知（飞书/钉钉/Webhook） |

### 工具

| 命令 | 说明 |
|------|------|
| [search](search.md) | 搜索已注册命令 |
| [completion](completion.md) | 生成 Shell 自动补全脚本 |

### 管理

| 命令 | 说明 |
|------|------|
| [install](install.md) | 系统包管理器 |
| [upgrade](upgrade.md) | 升级 opsxcli |
| [setup](setup.md) | 配置 LLM 提供商 |

### AI

| 命令 | 说明 |
|------|------|
| [agent](agent.md) | AI 运维助手 |
| [session](session.md) | AI 会话管理 |

---

## 使用示例

```bash
# 查看帮助
opsxcli --help
opsxcli <command> --help

# 系统监控
opsxcli sys
opsxcli free -h
opsxcli df -h

# 网络诊断
opsxcli net
opsxcli ping 8.8.8.8
opsxcli ss -tlnp
opsxcli ssl example.com

# SSH 连接
opsxcli ssh root@host
opsxcli ssh-config list

# 数据库操作
opsxcli mysql -u root -p "password" -h localhost
opsxcli mysqldump -d mydb -o backup.sql
opsxcli redis -h 127.0.0.1

# HTTP 工具
opsxcli curl https://api.example.com
opsxcli bench -c 50 -n 1000 https://api.example.com
opsxcli notify "部署完成" -t https://hook.example.com --type feishu

# Docker / K8s
opsxcli docker pull nginx:latest
opsxcli kubectl get pods -n default

# Agent 智能运维
opsxcli agent "内存使用率"
opsxcli agent "检查 192.168.1.100 的磁盘空间"
opsxcli agent -i                    # 交互模式
opsxcli session list                # 查看历史会话
```
