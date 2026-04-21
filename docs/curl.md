# curl

opsxcli curl — HTTP 请求工具

## 用法

`opsxcli curl [flags] <URL>`

## 说明

发送 HTTP 请求并显示响应。默认使用 GET 方法，支持自定义请求方法、请求头、POST 数据等。

## 选项

- `-X, --request <方法>`：指定 HTTP 请求方法（默认 GET）
- `-H, --header <头部>`：添加自定义请求头，可多次使用
- `-d, --data <数据>`：发送 POST 请求体数据
- `-o, --output <文件>`：将响应体保存到文件
- `-i, --include`：在输出中包含响应头
- `-I, --head`：发送 HEAD 请求，只显示响应头
- `-v, --verbose`：显示详细的请求和响应信息
- `-L, --location`：跟随 HTTP 重定向

## 示例

```bash
# GET 请求
opsxcli curl https://api.example.com

# POST 请求
opsxcli curl -X POST -d '{"name":"test"}' https://api.example.com

# 自定义请求头
opsxcli curl -H "Content-Type: application/json" -H "Authorization: Bearer token" https://api.example.com

# 保存响应到文件
opsxcli curl -o output.json https://api.example.com/data

# 显示响应头
opsxcli curl -i https://api.example.com

# HEAD 请求
opsxcli curl -I https://api.example.com

# 跟随重定向并显示详情
opsxcli curl -vL https://example.com
```
