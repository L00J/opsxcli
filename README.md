# opsxcli - 运维瑞士军刀 | 一站式命令行工具集

opsxcli 是一个面向运维和开发的集成化命令行工具集，内置数据库连接、网络调试、系统监控、文件传输等常用功能。

**无需切换多种客户端，一条命令即可操作 MySQL、Redis、SSH、HTTP 等服务，显著提升日常运维与调试效率。**

## ✨ 核心特性

- 🚀 **体积优化**: 15M (优化编译，相比22M减少32%)
- ⚡ **实时监控**: 2秒刷新，数据实时更新
- 🎨 **TUI界面**: 类似 htop/iftop 的终端可视化界面
- 📦 **网络工具替代**: 完全替代 iproute 和 net-tools 包
- 🔧 **busybox兼容**: 整合常用基础命令

## 🛠️ 完整命令列表

使用 `opsxcli <command> --help` 查看具体命令的帮助信息。

### 📁 文件 - 文件和目录操作
- **ls**: 列出目录内容
- **cat**: 显示文件内容
- **grep**: 文本搜索
- **vi**: 文本编辑器
- **cp**: 复制文件/目录
- **mv**: 移动/重命名文件
- **rm**: 删除文件/目录
- **mkdir**: 创建目录
- **tree**: 树形显示目录结构

### 💾 数据库 - MySQL, PostgreSQL, Redis
- **mysql**: MySQL数据库操作（支持交互式shell）
- **psql**: PostgreSQL数据库操作
- **redis**: Redis操作（支持单机和集群）

### 🖥️ 系统 - 进程和系统信息
- **ps**: 进程查看
- **top**: 进程实时监控
- **free**: 内存使用情况
- **df**: 磁盘空间使用
- **du**: 目录空间使用
- **uname**: 系统信息
- **hostname**: 主机名管理

### 🌐 网络 - SSH, Ping, 端口扫描等
- **ssh**: SSH连接、命令执行、文件传输、端口转发
- **ping**: 网络连通性测试
- **traceroute**: 路由追踪
- **telnet**: Telnet客户端
- **nc**: 网络连接工具（端口监听、内网反弹）
- **ss**: 网络连接状态查看（高性能，替代netstat）
- **nmap**: 网络端口扫描

### ⚙️ 网络配置 - 网络接口和路由管理
- **ifconfig**: 网络接口配置（兼容传统命令）
- **route**: 路由表管理
- **ip**: 现代网络配置工具

### 📦 压缩 - 归档和压缩工具
- **tar**: 归档工具
- **gzip**: GZIP压缩
- **unzip**: ZIP解压

### 🚀 服务 - HTTP/WebSocket/gRPC 服务
- **server**: HTTP/WebSocket/gRPC 服务端

### 🔧 工具 - 下载, HTTP请求, 安装
- **wget**: 文件下载（支持断点续传）
- **request**: 高级HTTP请求工具
- **install**: 工具安装脚本

### 📊 监控 - 系统监控, 网络监控 (2秒实时刷新)
- **sys**: 系统监控 TUI
  - CPU/内存/磁盘实时监控
  - 进程列表（支持多种排序）
  - 磁盘I/O统计
- **net**: 网络监控 TUI
  - 网络流量实时监控
  - 活跃连接TOP10
  - 连接状态统计

## 安装

### 快速下载（推荐）

自动检测平台并下载最新版本：

```bash
# 使用 wget
wget "https://gitee.com/opsx-tools/opsxcli/releases/download/latest/opsxcli-$(uname -s)-$(uname -m).tar.gz"

# 或使用 curl
curl -L -o opsxcli-$(uname -s)-$(uname -m).tar.gz \
  "https://gitee.com/opsx-tools/opsxcli/releases/download/latest/opsxcli-$(uname -s)-$(uname -m).tar.gz"
```

**注意**：
- Linux/macOS 用户下载 `.tar.gz` 文件
- Windows 用户请访问 [Releases 页面](https://gitee.com/opsx-tools/opsxcli/releases) 下载对应的 `.zip` 文件

**手动安装**

```bash
# Linux/macOS
tar -xzf opsxcli-*.tar.gz
chmod +x opsxcli
sudo mv opsxcli /usr/local/bin/

# Windows
# 解压 zip 文件后直接运行
```


## 使用示例

### SSH工具

```bash
# 交互式登录
opsxcli ssh root@172.16.1.123

# 执行命令
opsxcli ssh root@172.16.1.123 "ls -la"

# 上传文件
opsxcli ssh put /local/file.txt root@172.16.1.123:/remote/file.txt

# 下载文件
opsxcli ssh get root@172.16.1.123:/remote/file.txt /local/file.txt

# 端口转发
opsxcli ssh forward local 8080:localhost:80 root@172.16.1.123
```

### MySQL工具

```bash
# 交互式shell
opsxcli mysql -u root -p "password" -h localhost

# 执行SQL
opsxcli mysql -u root -p "password" -h localhost -e "SELECT VERSION()"
```

### Redis工具

```bash
# 交互式shell
opsxcli redis interactive -h 127.0.0.1 -a password

# 获取键值
opsxcli redis get mykey -h 127.0.0.1

# 设置键值
opsxcli redis set mykey "myvalue" -h 127.0.0.1
```

### 系统监控 (sys)

```bash
# 进入系统监控界面（2秒实时刷新）
opsxcli sys

# 快捷键操作：
# Tab: 切换视图 (概览/CPU/内存/磁盘/进程)
# C/M/D/T: 切换进程排序 (CPU/内存/磁盘IO/CPU时间)
# 1/2/3/4: 切换磁盘排序 (空间/Inodes/IO/名称)
# ↑↓: 选择进程
# q/ESC: 退出
```

### 网络监控 (net)

```bash
# 进入网络监控界面（2秒实时刷新）
opsxcli net

# 快捷键操作：
# Tab: 切换视图 (概览/连接/统计)
# q/ESC: 退出

# 显示内容：
# - 实时上传/下载速率
# - 活跃连接TOP10 (按流量排序)
# - 连接状态统计 (ESTABLISHED/TIME_WAIT/LISTEN等)
```

### 网络连接查看 (ss)

```bash
# 查看所有TCP和UDP连接
opsxcli ss -tunap

# 查看TCP状态统计（类似: ss -ant | awk '{++s[$1]} END {for(k in s) print k,s[k]}'）
# 用于查看本机并发和超时等TCP状态
opsxcli ss -ant --stats

# 查看TIME_WAIT状态的目标地址TOP 10
# 类似: ss -tan | awk '/TIME-WAIT/ {print $5}' | awk -F: '{print $1}' | sort | uniq -c | sort -rn | head -n5
opsxcli ss -tan --timewait

# 查看目标地址TOP 10
# 类似: opsxcli ss -an|awk '{print $5}'|awk -F: '{print $1}'|sort|egrep -o '[0-9]{1,3}(\.[0-9]{1,3}){3}'|uniq -c|sort -nr|head -n 10
opsxcli ss -an --top 10

# 只显示监听状态的连接
opsxcli ss -l

# 显示所有连接（包括监听）
opsxcli ss -a
```


## 📸 界面预览

### 系统监控 (sys)

![系统监控界面](docs/sys.jpeg)

### 网络监控 (net)

![网络监控界面](docs/net.jpeg)


## 许可证

MIT License
