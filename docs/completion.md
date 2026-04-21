# completion

opsxcli completion — 生成 Shell 自动补全脚本

## 用法

`opsxcli completion <bash|zsh|fish>`

## 说明

为 Bash、Zsh 或 Fish Shell 生成 opsxcli 的自动补全脚本。生成后需加载脚本或将其保存到 Shell 补全目录中以持久生效。

## 选项

无额外选项。

## 示例

生成 Bash 补全脚本：

```bash
opsxcli completion bash > /etc/bash_completion.d/opsxcli
```

生成 Zsh 补全脚本：

```bash
opsxcli completion zsh > "${fpath[1]}/_opsxcli"
```

生成 Fish 补全脚本：

```bash
opsxcli completion fish > ~/.config/fish/completions/opsxcli.fish
```

临时加载 Bash 补全（当前会话）：

```bash
source <(opsxcli completion bash)
```
