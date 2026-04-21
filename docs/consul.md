# consul

opsxcli consul — 服务发现与 Consul 注册

## 用法

`opsxcli consul [flags]`

## 说明

支持两种运行模式：K8s 模式（从 Kubernetes 集群发现服务并注册到 Consul）和云主机模式（从主机列表发现服务并注册到 Consul）。支持清理 Consul 中的失效实例。

## 选项

### 通用选项

- `-s, --service <地址>`：Consul 服务地址
- `-m, --metrics <路径>`：指标采集路径（默认 `/actuator/prometheus`）
- `--clean`：清理 Consul 中的失效服务实例
- `--insecure`：跳过 TLS 证书验证

### K8s 模式选项

- `-k, --kubeconfig <路径>`：kubeconfig 文件路径
- `--clear-cache`：清除缓存

### 云主机模式选项

- `--hosts <主机列表>`：云主机地址列表
- `--hosts-file <文件>`：从文件读取主机列表
- `--app-port <端口>`：应用端口（默认 8080）
- `--node-exp-port <端口>`：Node Exporter 端口（默认 9100）
- `--skip-node-exporter`：跳过 Node Exporter 注册

## 示例

```bash
# K8s 模式：注册服务到 Consul
opsxcli consul -s https://consul.example.com:8500

# K8s 模式：指定 kubeconfig
opsxcli consul -s https://consul.example.com:8500 -k ~/.kube/config

# K8s 模式：清理失效实例
opsxcli consul -s https://consul.example.com:8500 --clean

# 云主机模式：注册主机上的服务
opsxcli consul -s https://consul.example.com:8500 --hosts "192.168.1.10,192.168.1.11"

# 云主机模式：从文件读取主机列表
opsxcli consul -s https://consul.example.com:8500 --hosts-file hosts.txt

# 自定义指标路径和端口
opsxcli consul -s https://consul.example.com:8500 --hosts "10.0.0.1" -m /metrics --app-port 9090
```
