# kill

opsxcli kill — 终止进程

## 用法

`opsxcli kill [args...]`

## 说明

向指定进程发送信号，默认发送 TERM 信号以请求进程正常退出。所有参数直接转发给系统 `kill` 命令。

## 选项

此命令无自定义选项，以下为系统 `kill` 常用参数：

| 参数 | 说明 |
|------|------|
| `-s SIGNAL` | 指定要发送的信号 |
| `-l` | 列出所有可用信号名称 |
| `-9` | 发送 SIGKILL 信号，强制终止进程 |
| `-15` | 发送 SIGTERM 信号（默认），请求正常退出 |
| `-HUP` | 发送 SIGHUP 信号，常用于重载配置 |

## 示例

正常终止进程：

```bash
opsxcli kill 1234
```

强制终止进程：

```bash
opsxcli kill -9 1234
```

列出所有可用信号：

```bash
opsxcli kill -l
```

向进程发送 USR1 信号：

```bash
opsxcli kill -s USR1 1234
```
