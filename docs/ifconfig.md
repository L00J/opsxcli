# Ifconfig 工具

网络接口配置工具。

## 使用

```bash
# 显示所有接口
opsxcli ifconfig

# 显示特定接口
opsxcli ifconfig eth0

# 启用接口
opsxcli ifconfig eth0 up

# 禁用接口
opsxcli ifconfig eth0 down

# 设置 IP 地址
opsxcli ifconfig eth0 192.168.1.100

# 设置子网掩码
opsxcli ifconfig eth0 netmask 255.255.255.0
```

## 参数

- `up`: 启用接口
- `down`: 禁用接口
- `netmask`: 设置子网掩码
- `broadcast`: 设置广播地址
