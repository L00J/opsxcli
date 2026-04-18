# Wget 工具

文件下载工具，支持断点续传。

## 使用

```bash
# 基本下载
opsxcli wget http://www.example.com/test.sql

# 指定输出文件名
opsxcli wget -O test.sql http://www.example.com/test.sql

# 断点续传
opsxcli wget -c http://www.example.com/test.sql
```

## 参数

- `-O, --output`: 输出文件名
- `-c, --continue`: 断点续传
