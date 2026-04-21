# hostname

opsxcli hostname — 显示或设置主机名

## 用法

`opsxcli hostname [args...]`

## 说明

显示当前系统的主机名，或在具有相应权限时设置主机名。所有参数直接转发给系统 `hostname` 命令，行为与原生命令一致。

## 选项

此命令无自定义选项，以下为系统 `hostname` 常用参数：

| 参数 | 说明 |
|------|------|
| `-a` | 显示主机别名 |
| `-d` | 显示 DNS 域名 |
| `-f` | 显示完全限定域名（FQDN） |
| `-i` | 显示主机 IP 地址 |
| `-s` | 显示短主机名（截取第一个 `.` 之前的部分） |
| `-I` | 显示所有网络接口的 IP 地址 |

## 示例

显示当前主机名：

```bash
opsxcli hostname
```

显示完全限定域名：

```bash
opsxcli hostname -f
```

显示所有 IP 地址：

```bash
opsxcli hostname -I
```

设置主机名（需 root 权限）：

```bash
opsxcli hostname new-hostname
```
