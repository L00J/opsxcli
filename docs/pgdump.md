# pgdump

opsxcli pgdump — 导出 PostgreSQL 数据库

## 用法

`opsxcli pgdump [flags]`

## 说明

导出 PostgreSQL 数据库为 SQL 或 CSV 格式文件。Go 原生实现，不依赖系统 `pg_dump` 二进制。支持指定表导出、忽略表、仅导出结构或数据。

## 选项

| 标志 | 说明 |
|------|------|
| `-h, --host` | 主机地址 |
| `-P, --port` | 端口号 |
| `-U, --user` | 用户名 |
| `-W, --password` | 密码 |
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
opsxcli pgdump -h 127.0.0.1 -U postgres -W secret -d mydb -o backup.sql

# 仅导出指定表
opsxcli pgdump -d mydb --tables users,products -o partial.sql

# 导出为 CSV 格式
opsxcli pgdump -d mydb --format csv -o data/

# 仅导出表结构
opsxcli pgdump -d mydb --no-data -o schema.sql
```
