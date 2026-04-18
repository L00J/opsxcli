# Nmap 工具

网络端口扫描工具，支持端口范围扫描和服务识别。

## 基本使用

```bash
# 扫描单个端口
opsxcli nmap localhost -p 80

# 扫描多个端口
opsxcli nmap localhost -p 80,443,22

# 扫描端口范围
opsxcli nmap localhost -p 22-9999

# 快速扫描常用端口
opsxcli nmap 192.168.1.1 -p 1-1000 -T 500ms -v
```

## 参数

- `-p, --ports`: 要扫描的端口范围（如：80,443 或 1-1000，默认: 1-1000）
- `-T, --timeout`: 连接超时时间（默认: 3s）
- `-v, --verbose`: 显示详细信息
