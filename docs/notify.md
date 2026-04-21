# notify

opsxcli notify — 发送告警通知

## 用法

`opsxcli notify <message> [flags]`

## 说明

发送告警通知到飞书、钉钉或通用 Webhook。支持多种通知类型和告警级别，可自动适配不同平台的消息格式。通过签名密钥确保消息安全。

## 选项

| 标志 | 说明 |
|------|------|
| `-t, --target` | 通知目标 URL（必填） |
| `--type` | 通知类型（`webhook`/`feishu`/`dingtalk`，默认 `webhook`） |
| `--title` | 消息标题（默认 `OpsXCLI 通知`） |
| `-l, --level` | 告警级别（`info`/`warning`/`error`/`critical`，默认 `info`） |
| `--at` | @用户列表（逗号分隔） |
| `--secret` | 签名密钥 |

## 示例

```bash
# 发送飞书通知
opsxcli notify "服务器 CPU 使用率超过 90%" -t https://open.feishu.cn/open-apis/bot/v2/hook/xxx --type feishu

# 发送钉钉告警
opsxcli notify "磁盘空间不足" -t https://oapi.dingtalk.com/robot/send?access_token=xxx --type dingtalk -l error

# 发送通用 Webhook 通知（带签名）
opsxcli notify "部署完成" -t https://hooks.example.com/ops --secret my-secret-key --title "部署通知"

# 严重告警并 @指定用户
opsxcli notify "数据库连接异常" -t https://open.feishu.cn/xxx --type feishu -l critical --at "user1,user2"
```
