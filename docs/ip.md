# ip

opsxcli ip — 显示网络信息

## 用法

`opsxcli ip <子命令>`

## 说明

显示网络配置信息，包括 IP 地址、网络接口和路由表。此命令为只读显示，不支持 set/add/del 等修改操作。

## 子命令

- `addr`：显示 IP 地址信息
- `link`：显示网络接口信息
- `route`：显示路由表信息

## 选项

无。

## 示例

```bash
# 显示 IP 地址
opsxcli ip addr

# 显示网络接口
opsxcli ip link

# 显示路由表
opsxcli ip route
```
