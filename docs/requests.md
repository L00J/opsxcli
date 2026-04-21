# Request 工具

高级HTTP请求工具，支持GET、POST等方法。

## 使用

```bash
# GET请求
opsxcli request https://api.example.com/users

# POST请求
opsxcli request -X POST -d "name=test" https://api.example.com/users

# 设置请求头
opsxcli request -H "Authorization: Bearer token" https://api.example.com/users

# 保存响应到文件
opsxcli request -o response.json https://api.example.com/users
```

## 参数

- `-X, --method`: HTTP方法（默认: GET）
- `-H, --header`: HTTP头（可多次使用）
- `-d, --data`: 请求体数据
- `-o, --output`: 输出到文件
