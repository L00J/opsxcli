# opsxcli - 运维瑞士军刀 | 一站式命令行工具集

opsxcli 是一个面向运维和开发的集成化命令行工具集,内置数据库连接、网络调试、系统监控、文件传输等常用功能。

**无需切换多种客户端，一条命令即可操作 MySQL、Redis、SSH、HTTP 等服务，显著提升日常运维与调试效率。**

## ✨ 核心特性

- 🚀 **体积优化**: 15M (优化编译，相比22M减少32%)
- ⚡ **实时监控**: 2秒刷新，数据实时更新
- 🎨 **TUI界面**: 类似 htop/iftop 的终端可视化界面
- 📦 **网络工具替代**: 完全替代 iproute 和 net-tools 包
- 🔧 **busybox兼容**: 整合常用基础命令

## 🛠️ 完整命令列表

使用 `opsxcli <command> --help` 查看具体命令的帮助信息。

### 📁 文件操作
- **ls**, **cat**, **grep**, **vi**, **vim**: 查看和搜索
- **more**, **less**, **head**, **tail**: 分页和预览
- **cp**, **mv**, **rm**: 复制、移动、删除
- **mkdir**, **rmdir**, **touch**: 目录和文件创建
- **tree**: 树形显示目录结构
- **chmod**, **chown**, **ln**: 权限和链接管理
- **awk**, **sed**: 文本处理

### 💾 数据库工具
- **mysql**: MySQL 交互式 Shell 和命令执行
- **psql**: PostgreSQL 数据库操作
- **redis**: Redis 单机/集群操作

### 🖥️ 系统工具
- **ps**, **top**, **kill**, **pstree**: 进程管理
- **free**, **df**, **du**: 资源查看
- **uname**, **hostname**, **whoami**, **id**: 系统信息
- **mount**, **umount**: 挂载管理
- **date**, **sleep**, **watch**: 时间和定时

### 🌐 网络工具
- **ssh**: SSH 连接、命令执行、文件传输、端口转发
- **ping**, **traceroute**: 连通性和路由测试
- **telnet**, **nc**: Telnet 和端口工具
- **ss**, **netstat**: 连接状态查看
- **nmap**: 端口扫描
- **ftp**, **tftp**: 文件传输

### ⚙️ 网络配置
- **ifconfig**, **route**, **ip**: 接口和路由管理

### 📦 压缩工具
- **tar**, **gzip**, **unzip**: 归档和压缩

### 🚀 服务端
- **server**: HTTP/WebSocket/gRPC 服务

### 🔧 实用工具
- **wget**: 文件下载（支持断点续传）
- **request**: HTTP 请求工具
- **install**: 安装脚本
- **upgrade**: 升级 opsxcli 到最新版本

### 🐳 Docker 工具
- **docker pull**: Docker 镜像拉取（自动加速、多源极速、并发下载、断点续传）

### ☸️ Kubernetes 工具
- **kubectl**: kubectl 命令行代理
- **consul**: 从 K8s 集群自动发现服务并注册到 Consul
- **kubernetes**: K8s 资源管理（YAML 生成、资源操作）

### 📊 监控 TUI (2秒实时刷新)
- **sys**: 系统监控（CPU/内存/磁盘/进程）
- **net**: 网络监控（流量/连接/状态统计）

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

### 手动安装

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

### Docker 镜像管理

```bash
# 拉取单个镜像（自动加速）
opsxcli docker pull nginx:latest

# 拉取多个镜像（并发）
opsxcli docker pull nginx:latest redis:alpine mysql:8.0

# 使用自定义镜像源
opsxcli docker pull nginx:latest -r docker.aityp.com -r docker.1ms.run

# 设置并发数
opsxcli docker pull nginx redis mysql -c 5
```

### Consul 服务注册

```bash
# 从 K8s 集群注册服务到 Consul
opsxcli consul -s https://consul.example.com:8500 -m /actuator/prometheus

# 指定 kubeconfig 路径
opsxcli consul -k /path/to/kubeconfig -s https://consul.example.com:8500 -m /metrics

# 清理失效实例
opsxcli consul -s https://consul.example.com:8500 --clean

# 清除缓存后重新扫描
opsxcli consul -s https://consul.example.com:8500 -m /actuator/prometheus --clear-cache
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

# 查看TCP状态统计
opsxcli ss -ant --stats

# 查看TIME_WAIT状态的目标地址TOP 10
opsxcli ss -tan --timewait

# 查看目标地址TOP 10
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
