# Server 工具

启动一个简单的HTTP文件服务器，用于在指定目录提供文件服务。

## 使用

```bash
# 在8000端口启动，服务当前目录
opsxcli server

# 指定端口
opsxcli server -p 8080

# 指定服务目录
opsxcli server -d /tmp

# 指定端口和目录
opsxcli server -p 9000 -d /var/www
```

## 参数

- `-p, --port`: 服务器端口（默认: 8000）
- `-d, --directory`: 服务目录（默认: 当前目录）

## 功能特性

- 自动记录访问日志
- 支持优雅关闭（Ctrl+C）
- 类似 `python3 -m http.server` 的功能

## 示例

```bash
# 启动服务器
opsxcli server -p 8000

# 在另一个终端访问
curl http://localhost:8000
```
