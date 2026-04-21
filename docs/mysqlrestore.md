# mysqlrestore

opsxcli mysqlrestore — 导入 SQL 文件到 MySQL 数据库

## 用法

`opsxcli mysqlrestore [flags]`

## 说明

将 SQL 文件导入到指定的 MySQL 数据库。支持从 `mysqldump` 导出的 `.sql` 文件恢复数据。执行前会校验连接和文件有效性。

## 选项

| 标志 | 说明 |
|------|------|
| `-h, --host` | 主机地址 |
| `-P, --port` | 端口号 |
| `-u, --user` | 用户名 |
| `-p, --password` | 密码 |
| `-d, --database` | 目标数据库名 |
| `-i, --input` | 输入的 SQL 文件路径 |

## 示例

```bash
# 导入 SQL 文件到数据库
opsxcli mysqlrestore -h 127.0.0.1 -u root -p secret -d mydb -i backup.sql

# 从远程主机恢复
opsxcli mysqlrestore -h db.example.com -P 3306 -u admin -p pass -d mydb -i dump.sql

# 恢复到新数据库
opsxcli mysqlrestore -h 127.0.0.1 -u root -p secret -d mydb_copy -i mydb_backup.sql
```
