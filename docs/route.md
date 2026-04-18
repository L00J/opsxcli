# Route 工具

路由表管理工具。

## 使用

```bash
# 显示路由表
opsxcli route

# 添加默认网关
opsxcli route add default gw 192.168.1.1

# 添加目标网络路由
opsxcli route add -net 10.0.0.0/8 gw 192.168.1.1

# 删除路由
opsxcli route del default

# 删除目标网络路由
opsxcli route del -net 10.0.0.0/8
```

## 参数

- `add`: 添加路由
- `del`: 删除路由
- `-net`: 目标网络
- `gw`: 网关地址
- `netmask`: 子网掩码
