# search

opsxcli search — 模糊搜索命令

## 用法

`opsxcli search <keyword> [keyword...]`

## 说明

模糊搜索所有已注册命令，支持多关键词 AND 匹配。输入多个关键词时，只返回同时匹配所有关键词的命令。用于快速查找不熟悉的命令。

## 选项

无自定义选项。可使用全局 `--output json` 标志以 JSON 格式输出结果。

## 示例

搜索单个关键词：

```bash
opsxcli search 文件
```

搜索多个关键词（AND 匹配）：

```bash
opsxcli search ssl 证书
```

以 JSON 格式输出搜索结果：

```bash
opsxcli search --output json 网络
```
