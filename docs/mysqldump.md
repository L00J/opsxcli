# mysqldump

opsxcli mysqldump — 导出 MySQL 数据库

## 用法

`opsxcli mysqldump [flags]`

## 说明

导出 MySQL 数据库为 SQL 或 CSV 格式文件。Go 原生实现，不依赖系统 `mysqldump` 二进制。支持指定表导出、忽略表、仅导出结构或数据等灵活选项。

## 选项

| 标志 | 说明 |
|------|------|
| `-h, --host` | 主机地址 |
| `-P, --port` | 端口号 |
| `-u, --user` | 用户名 |
| `-p, --password` | 密码 |
| `-d, --database` | 数据库名 |
| `-o, --output` | 输出文件路径 |
| `--format` | 导出格式 (`sql`/`csv`，默认 `sql`) |
| `--tables` | 指定导出的表（逗号分隔） |
| `--ignore-tables` | 忽略的表（逗号分隔） |
| `--no-data` | 仅导出结构 |
| `--no-schema` | 仅导出数据 |

## 示例

```bash
# 导出整个数据库
opsxcli mysqldump -h 127.0.0.1 -u root -p secret -d mydb -o backup.sql

# 仅导出指定表为 CSV
opsxcli mysqldump -d mydb --tables users,orders --format csv -o data/

# 仅导出表结构
opsxcli mysqldump -d mydb --no-data -o schema.sql

# 忽略日志表导出
opsxcli mysqldump -d mydb --ignore-tables access_log,error_log -o full.sql
```
