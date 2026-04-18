# IP 工具

网络配置工具（iproute2）。

## 使用

```bash
# 显示所有接口
opsxcli ip addr

# 显示接口详情
opsxcli ip addr show eth0

# 启用接口
opsxcli ip link set eth0 up

# 禁用接口
opsxcli ip link set eth0 down

# 设置 IP 地址
opsxcli ip addr add 192.168.1.100/24 dev eth0

# 删除 IP 地址
opsxcli ip addr del 192.168.1.100/24 dev eth0

# 显示路由表
opsxcli ip route

# 添加默认网关
opsxcli ip route add default via 192.168.1.1

# 显示网络邻居
opsxcli ip neigh
```

## 子命令

- `addr`: IP 地址管理
- `link`: 网络接口管理
- `route`: 路由表管理
- `neigh`: 邻居 ARP 表
