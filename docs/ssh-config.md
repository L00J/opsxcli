# ssh-config

opsxcli ssh-config — 管理 SSH 配置文件

## 用法

`opsxcli ssh-config [command] [flags]`

## 说明

读取并解析 SSH 配置文件，提供主机列表查看和详细配置查询功能。默认读取 `~/.ssh/config`。

## 持久选项

| 标志 | 说明 |
|------|------|
| `--config` | 指定 SSH 配置文件路径（默认 `~/.ssh/config`） |

## 子命令

### list

列出所有已配置的主机，显示别名、地址、用户、端口和私钥路径。

```bash
opsxcli ssh-config list
```

指定配置文件：

```bash
opsxcli ssh-config list --config /etc/ssh/ssh_config
```

### show

显示指定主机的详细配置信息。

```bash
opsxcli ssh-config show myserver
```

## 示例

列出所有主机：

```bash
opsxcli ssh-config list
```

查看特定主机配置：

```bash
opsxcli ssh-config show prod-web01
```

使用自定义配置文件：

```bash
opsxcli ssh-config --config ~/.ssh/config.work list
```
