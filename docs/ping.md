# Ping 工具

网络连通性测试工具。

## 使用

```bash
# 持续ping
opsxcli ping baidu.com

# 指定次数
opsxcli ping -c 5 baidu.com

# 设置间隔和超时
opsxcli ping -i 2s -W 5s baidu.com
```

## 参数

- `-c, --count`: 发送次数（-1表示持续，默认持续）
- `-i, --interval`: 发送间隔（默认: 1s）
- `-W, --timeout`: 超时时间（默认: 3s）
