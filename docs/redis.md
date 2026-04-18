# Redis 工具

Redis操作工具，支持单机和集群模式。

## 交互式模式

```bash
# 进入交互式Redis shell
opsxcli redis -h 127.0.0.1 -a password
opsxcli redis -h 127.0.0.1 -p 7001 -c -a "password"  # 集群模式

# 交互模式功能：
# - TAB 键自动补全
# - 上下箭头键历史记录
# - help 查看帮助
# - exit 或 quit 退出
```

## 基本操作

```bash
# 获取键值
opsxcli redis get mykey -h 127.0.0.1

# 设置键值
opsxcli redis set mykey "myvalue" -h 127.0.0.1
opsxcli redis set mykey "myvalue" -e 1h  # 设置过期时间
```

## 集群模式

```bash
# 启用集群模式
opsxcli redis -h 127.0.0.1 -p 7001 -c -a "password"

# 指定多个集群节点
opsxcli redis --cluster --addrs "127.0.0.1:7001,127.0.0.1:7002" -a "password"
```

## 参数

- `-h, --host`: Redis主机地址（默认: 127.0.0.1）
- `-p, --port`: Redis端口（默认: 6379）
- `-a, --password`: Redis密码
- `-d, --db`: 数据库编号（默认: 0）
- `-e, --expire`: 过期时间（如: 1h, 30m, 60s）
- `-c, --cluster`: 集群模式
- `--addrs`: 集群地址列表（逗号分隔）
