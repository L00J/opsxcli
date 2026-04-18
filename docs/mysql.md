# MySQL 工具

MySQL数据库操作工具，支持交互式shell和SQL执行。

## 交互式模式

```bash
# 进入交互式MySQL shell
opsxcli mysql -u root -p "password" -h localhost

# 交互模式功能：
# - TAB 键自动补全
# - 上下箭头键历史记录
# - 多行 SQL（以 ; 结尾执行）
# - help; 或 \h 查看帮助
# - exit; 或 \q 退出
```

## 执行SQL

```bash
opsxcli mysql -u root -p "password" -h localhost -e "SELECT VERSION()"
opsxcli mysql -u root -p "password" -h localhost -d mydb -e "SHOW TABLES"
```

## 参数

- `-h, --host`: 主机地址（默认: 127.0.0.1）
- `-P, --port`: 端口（默认: 3306）
- `-u, --user`: 用户名（默认: root）
- `-p, --password`: 密码
- `-d, --database`: 数据库名
- `-e, --execute`: 执行SQL语句
