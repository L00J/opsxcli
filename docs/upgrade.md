# Upgrade 工具

自动升级 opsxcli 到最新版本。

## 使用

```bash
# 检查并升级到最新版本
opsxcli upgrade

# 查看当前版本
opsxcli upgrade --version

# 指定版本升级
opsxcli upgrade -v 1.2.3

# 查看可升级版本
opsxcli upgrade --check
```

## 参数

- `-v, --version`: 指定版本号
- `-c, --check`: 只检查，不升级
- `--beta`: 升级到测试版
- `--force`: 强制升级

## 功能特性

- 自动检测当前系统和架构
- 从 Gitee Releases 获取最新版本
- 下载并替换当前执行文件
- 保留所有配置和数据
- 升级前自动备份

## 示例

```bash
# 检查最新版本
opsxcli upgrade --check

# 升级到最新版
opsxcli upgrade

# 升级到指定版本
opsxcli upgrade -v 1.2.3
```
