# logs

opsxcli logs — 分析日志文件，生成统计报告

## 用法

`opsxcli logs <file> [flags]`

## 说明

分析 Nginx、Apache、Syslog 格式的日志文件，自动识别格式并生成统计报告，包括访问量 Top N、状态码分布、慢请求排名、流量统计等。支持 `.gz` 压缩文件和大文件流式处理。

## 选项

| 标志 | 说明 |
|------|------|
| `-t, --type` | 日志格式（`nginx`/`apache`/`syslog`/`auto`，默认 `auto`） |
| `--top` | Top N 排名数量（默认 `10`） |
| `--slow` | 慢请求阈值，单位毫秒（默认 `1000`） |
| `--since` | 开始时间 |
| `--until` | 结束时间 |
| `-f, --format` | 输出格式（`text`/`json`，默认 `text`） |
| `--max-lines` | 最大处理行数（默认 `1000000`） |

## 示例

```bash
# 自动检测格式并分析 Nginx 日志
opsxcli logs /var/log/nginx/access.log

# 指定格式，显示 Top 20 URI，慢请求阈值 500ms
opsxcli logs access.log -t nginx --top 20 --slow 500

# 分析时间范围内的日志，JSON 输出
opsxcli logs access.log --since "2024-01-01" --until "2024-01-31" -f json

# 分析压缩日志文件
opsxcli logs /var/log/nginx/access.log.1.gz -t nginx
```
