# bench

opsxcli bench — HTTP 压力测试工具

## 用法

`opsxcli bench <url> [flags]`

## 说明

HTTP 压力测试工具，支持并发控制和持续时间两种模式。可自定义请求方法、请求体、请求头等参数，输出详细的性能统计报告，包括 QPS、延迟分布、状态码统计等。

## 选项

| 标志 | 说明 |
|------|------|
| `-c, --concurrency` | 并发数（默认 `10`） |
| `-n, --requests` | 总请求数（默认 `100`） |
| `-d, --duration` | 持续时间（如 `30s`、`1m`） |
| `-m, --method` | HTTP 方法（默认 `GET`） |
| `-b, --body` | 请求体 |
| `-H, --header` | 自定义请求头，可多次指定（格式 `Key: Value`） |
| `--timeout` | 单请求超时（默认 `30s`） |
| `--json` | JSON 格式输出 |
| `--insecure` | 跳过 TLS 证书验证 |

## 示例

```bash
# 100 并发发送 1000 个 GET 请求
opsxcli bench http://localhost:8080 -c 100 -n 1000

# 持续压测 30 秒
opsxcli bench https://api.example.com -d 30s -c 50

# POST 请求带自定义头
opsxcli bench http://api/test -m POST -b '{"key":"val"}' -H "Content-Type: application/json"

# JSON 格式输出并跳过 TLS 验证
opsxcli bench https://staging.local -c 20 -n 500 --json --insecure
```
