# Curl 工具

HTTP 请求工具。

## 使用

```bash
# GET 请求
opsxcli curl https://api.example.com

# POST 请求
opsxcli curl -X POST -d "name=test" https://api.example.com

# 设置请求头
opsxcli curl -H "Content-Type: application/json" https://api.example.com

# 保存响应
opsxcli curl -o output.json https://api.example.com/data

# 显示响应头
opsxcli curl -i https://api.example.com

# 显示详细请求过程
opsxcli curl -v https://api.example.com

# 跟随重定向
opsxcli curl -L https://api.example.com
```

## 参数

- `-X, --method`: HTTP 方法
- `-H, --header`: 请求头
- `-d, --data`: 请求体数据
- `-o, --output`: 输出到文件
- `-i, --include`: 显示响应头
- `-v, --verbose`: 详细输出
- `-L, --location`: 跟随重定向
- `-k, --insecure`: 忽略 SSL 证书
