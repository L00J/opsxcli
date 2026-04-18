<!--
 * @Author: Logan.Li
 * @Gitee: https://gitee.com/attacker
 * @email: admin@attacker.club
 * @Date: 2025-12-15 03:15:28
 * @LastEditTime: 2025-12-15 22:19:49
 * @Description: 
-->
# SSH 工具

SSH连接、命令执行、文件传输、端口转发。

## 连接

```bash
# 交互式登录
opsxcli ssh root@172.16.1.123
opsxcli ssh root@172.16.1.123 -i ~/.ssh/id_rsa
opsxcli ssh root@172.16.1.123 -P  # 提示输入密码

# 执行命令
opsxcli ssh root@172.16.1.123 "ls -la" -i ~/.ssh/id_rsa
```

## 文件传输

```bash
# 上传文件
opsxcli ssh put /local/file.txt root@172.16.1.123:/remote/file.txt

# 下载文件
opsxcli ssh get root@172.16.1.123:/remote/file.txt /local/file.txt
```

## 端口转发

```bash
# 本地端口转发
opsxcli ssh forward local 8080:localhost:80 root@172.16.1.123

# 远程端口转发
opsxcli ssh forward remote 8080:localhost:80 root@172.16.1.123

# 动态端口转发/SOCKS代理
opsxcli ssh forward dynamic 1080 root@172.16.1.123
```

## 参数

- `-i, --key`: SSH私钥路径
- `-p, --port`: SSH端口（默认: 22）
- `-P, --password`: 密码（不指定则提示输入）
