# Kubernetes 工具

K8s 资源管理工具。

## 使用

```bash
# 生成 YAML
opsxcli kubernetes gen deployment --name nginx --image nginx:latest

# 生成 Service
opsxcli kubernetes gen service --name nginx --port 80

# 资源操作
opsxcli kubernetes apply -f nginx.yaml
opsxcli kubernetes delete -f nginx.yaml

# 查看资源
opsxcli kubernetes list pods
opsxcli kubernetes list services
```

## 子命令

### gen - 生成 YAML

```bash
# 生成 Deployment
opsxcli kubernetes gen deployment --name nginx --image nginx:latest

# 生成 ConfigMap
opsxcli kubernetes gen configmap --name app-config --from-file=config.yaml

# 生成 Secret
opsxcli kubernetes gen secret --name db-credentials --from-literal=password=xxx
```

### apply - 应用资源

```bash
opsxcli kubernetes apply -f deployment.yaml
opsxcli kubernetes apply -f .  # 应用目录下所有 YAML
```

### delete - 删除资源

```bash
opsxcli kubernetes delete deployment nginx
opsxcli kubernetes delete -f deployment.yaml
```

### list - 列出资源

```bash
opsxcli kubernetes list pods
opsxcli kubernetes list services
opsxcli kubernetes list deployments
opsxcli kubernetes list all
```
