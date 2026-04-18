# PostgreSQL 工具

PostgreSQL数据库操作工具，支持交互式shell和SQL执行。

## 交互式模式

```bash
# 进入交互式PostgreSQL shell
opsxcli psql -h localhost -P 5432 -U postgres -W password

# 在交互模式中：
# - 按 TAB 键自动补全 SQL 命令
# - 支持命令历史记录（上下箭头键）
# - 支持多行 SQL（以 ; 结尾执行）
# - 支持 psql 特殊命令（\l, \dt, \d 等）
# - 输入 'exit' 或 '\q' 退出

postgres=# SELECT version();
postgres=# \l  # 列出所有数据库
postgres=# \dt # 列出表
```

## 执行SQL

```bash
opsxcli psql -h localhost -U postgres -W password -c "SELECT version()"
opsxcli psql -h localhost -U postgres -W password -d mydb -c "\dt"
opsxcli psql -h localhost -U postgres -c "\l"  # 列出所有数据库
```

## 参数

- `-h, --host`: 主机地址（默认: 127.0.0.1）
- `-P, --port`: 端口（默认: 5432）
- `-U, --user`: 用户名（默认: postgres）
- `-W, --password`: 密码
- `-d, --database`: 数据库名
- `-c, --execute`: 执行SQL语句
