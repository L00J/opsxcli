# WebSearch 工具

网络搜索工具。

## 使用

```bash
# 搜索关键词
opsxcli websearch "golang tutorial"

# 搜索多个引擎
opsxcli websearch -e baidu -e google "kubernetes"

# 指定结果数量
opsxcli websearch -n 10 "docker pull 加速"

# 保存 HTML 结果
opsxcli websearch -s -o results.html "opsxcli"
```

## 参数

- `-e, --engine`: 搜索引擎（baidu, google, bing）
- `-n, --num`: 结果数量
- `-s, --save`: 保存 HTML 结果
- `-o, --output`: 输出文件

## 支持的搜索引擎

- `baidu`: 百度搜索
- `google`: Google 搜索
- `bing`: Bing 搜索
