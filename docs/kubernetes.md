# kubernetes

opsxcli kubernetes — Kubernetes 集群管理工具

## 用法

`opsxcli kubernetes <子命令> [flags]`

## 说明

提供 Kubernetes 集群的健康检查、资源管理和 YAML 操作功能。

## 选项

- `-k, --kubeconfig <路径>`：指定 kubeconfig 文件路径

## 子命令

### check — 健康检查

对 Kubernetes 集群执行健康检查。

### resource — 资源管理

管理 Kubernetes 集群资源。

### yaml — YAML 操作

处理 Kubernetes YAML 资源文件。

## 示例

```bash
# 集群健康检查
opsxcli kubernetes check

# 指定 kubeconfig 进行健康检查
opsxcli kubernetes check -k /path/to/kubeconfig

# 资源管理
opsxcli kubernetes resource

# YAML 操作
opsxcli kubernetes yaml
```
