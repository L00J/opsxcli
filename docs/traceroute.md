# Traceroute 工具

路由追踪工具，显示数据包到目标主机的路由路径。

## 使用

```bash
opsxcli traceroute baidu.com
opsxcli traceroute baidu.com -m 30 -s 60
```

## 参数

- `-m, --max-hops`: 最大跳数（默认: 30）
- `-s, --packet-size`: 数据包大小，字节（默认: 60）
