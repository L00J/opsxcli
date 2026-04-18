# Install 工具

自动检测Linux系统类型并使用对应的包管理器安装软件。

## 使用

```bash
# 安装软件包
opsxcli install mysql
opsxcli install nginx
opsxcli install docker
```

## 支持的系统

### 使用 yum 的系统
- RHEL (Red Hat Enterprise Linux)
- CentOS
- Rocky Linux
- AlmaLinux
- Amazon Linux
- 阿里云 Linux 2

### 使用 dnf 的系统
- Fedora
- 阿里云 Linux 3/4

### 使用 apt-get 的系统
- Debian
- Ubuntu

## 工作原理

1. 自动检测系统类型（读取 `/etc/os-release` 等文件）
2. 根据系统类型选择对应的包管理器
3. 执行安装命令

## 示例

```bash
# 在 CentOS 上安装 MySQL
opsxcli install mysql
# 实际执行: yum install -y mysql

# 在 Ubuntu 上安装 Nginx
opsxcli install nginx
# 实际执行: apt-get install -y nginx

# 在阿里云 Linux 3 上安装 Docker
opsxcli install docker
# 实际执行: dnf install -y docker
```

## 注意事项

- 需要 root 权限执行
- 仅支持 Linux 系统
- 自动使用 `-y` 参数，无需确认
