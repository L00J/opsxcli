# Kubectl 工具

Kubernetes kubectl 命令行代理。

## 使用

```bash
# 获取 Pod 列表
opsxcli kubectl get pods

# 获取所有资源
opsxcli kubectl get all

# 查看 Pod 详情
opsxcli kubectl describe pod nginx-xxx

# 查看日志
opsxcli kubectl logs nginx-xxx

# 切换上下文
opsxcli kubectl config use-context prod-cluster

# 列出上下文
opsxcli kubectl config get-contexts
```

## 常用命令

```bash
# 命名空间操作
opsxcli kubectl get namespaces
opsxcli kubectl get pods -n kube-system

# Deployment 操作
opsxcli kubectl get deployments
opsxcli kubectl scale deployment/nginx --replicas=3

# Service 操作
opsxcli kubectl get services
opsxcli kubectl expose deployment/nginx --port=80 --type=LoadBalancer

# 删除资源
opsxcli kubectl delete pod nginx-xxx
```

## 说明

此命令是 kubectl 的代理，会将命令转发给 kubectl 执行。
