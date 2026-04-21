# kubectl

opsxcli kubectl — Kubernetes 资源管理工具（原生 Go 实现）

## 用法

`opsxcli kubectl [子命令] [flags]`

## 说明

使用原生 Go Kubernetes API 实现的资源管理工具，不是 kubectl 二进制的封装。支持常用的 Kubernetes 资源操作。

## 公共选项

- `-k, --kubeconfig <路径>`：指定 kubeconfig 文件路径
- `-n, --namespace <命名空间>`：指定命名空间

## 子命令

### get — 获取资源列表

- `-A, --all-namespaces`：显示所有命名空间的资源
- `-o, --output <格式>`：输出格式（如 wide、yaml、json）

### describe — 查看资源详情

### logs — 查看 Pod 日志

- `-f, --follow`：持续跟踪日志输出
- `--tail N`：显示最后 N 行日志

### delete — 删除资源

### scale — 扩缩副本数

- `--replicas N`：指定目标副本数

### rollout status — 查看滚动更新状态

### apply — 应用资源配置

### exec — 在容器中执行命令

- `-c, --container <名称>`：指定容器
- `-i, --stdin`：保持标准输入打开
- `-t, --tty`：分配伪终端

### cp — 复制文件

- `-c, --container <名称>`：指定容器

### top — 查看资源使用

- 子命令：`pods`、`nodes`

## 示例

```bash
# 获取 Pod 列表
opsxcli kubectl get pods -n default

# 获取所有命名空间的 Pod
opsxcli kubectl get pods -A

# 查看 Pod 日志
opsxcli kubectl logs nginx-xxx -n default

# 持续跟踪日志
opsxcli kubectl logs -f nginx-xxx --tail 100

# 扩缩副本
opsxcli kubectl scale deployment/nginx --replicas 3

# 在容器中执行命令
opsxcli kubectl exec -it nginx-xxx -c main -- /bin/sh

# 查看 Pod 资源使用
opsxcli kubectl top pods
```
