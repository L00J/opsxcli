# ssl

opsxcli ssl — 查询 SSL/TLS 证书信息

## 用法

`opsxcli ssl <domain> [flags]`

## 说明

连接指定域名，获取并展示 SSL/TLS 证书信息，包括颁发者、有效期、域名等。支持证书链查看和到期预警。

## 选项

| 标志 | 说明 |
|------|------|
| `-p, --port` | 端口号（默认 443） |
| `--chain` | 显示完整证书链 |
| `--warn` | 到期警告天数（默认 30） |
| `--json` | 以 JSON 格式输出 |
| `--timeout` | 连接超时时间（默认 10s） |

## 示例

查询域名证书信息：

```bash
opsxcli ssl example.com
```

指定端口查询：

```bash
opsxcli ssl example.com -p 8443
```

查看完整证书链：

```bash
opsxcli ssl example.com --chain
```

JSON 格式输出并设置超时：

```bash
opsxcli ssl example.com --json --timeout 5s
```
