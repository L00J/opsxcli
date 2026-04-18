# Consul 工具

K8s 服务发现并注册到 Consul。

## 使用

```bash
# 注册 K8s 服务到 Consul
opsxcli consul -s https://consul.example.com:8500

# 注册带指标路径
opsxcli consul -s https://consul.example.com:8500 -m /actuator/prometheus

# 清理失效实例
opsxcli consul -s https://consul.example.com:8500 --clean

# 预览注册内容
opsxcli consul -s https://consul.example.com:8500 --dry-run
```

## 参数

- `-s, --server`: Consul 服务器地址
- `-m, --metrics-path`: 指标路径（默认: /health）
- `--clean`: 清理失效的服务实例
- `--dry-run`: 预览模式，不执行注册

## 工作原理

1. 连接 Kubernetes API 获取所有 Services
2. 将 K8s Service 转换为 Consul Service
3. 注册到 Consul Catalog
4. 定期健康检查

## 示例

```bash
# 注册生产环境服务
opsxcli consul -s https://consul.opsxcli.com:8500 -m /health

# 清理失效实例（定时任务）
opsxcli consul -s https://consul.opsxcli.com:8500 --clean
```
