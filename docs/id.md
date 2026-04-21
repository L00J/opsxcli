# id

opsxcli id — 显示用户和组信息

## 用法

`opsxcli id [args...]`

## 说明

显示当前用户（或指定用户）的用户 ID（UID）、组 ID（GID）以及所属的附加组信息。所有参数直接转发给系统 `id` 命令。

## 选项

此命令无自定义选项，以下为系统 `id` 常用参数：

| 参数 | 说明 |
|------|------|
| `-u` | 仅显示有效 UID |
| `-g` | 仅显示有效 GID |
| `-G` | 显示所有附属组 ID |
| `-n` | 显示名称而非数字（与 -u/-g/-G 配合使用） |
| `-r` | 显示真实 ID 而非有效 ID |

## 示例

显示当前用户的完整信息：

```bash
opsxcli id
```

仅显示当前用户的 UID：

```bash
opsxcli id -u
```

显示指定用户的信息：

```bash
opsxcli id root
```

以名称方式显示所有附属组：

```bash
opsxcli id -Gn
```
