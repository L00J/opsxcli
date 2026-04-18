# Bash 命令风险分类 - 测试用例

## 📋 测试说明

本文档包含所有命令风险分类的测试用例,用于验证 `bash_analyzer.go` 的准确性。

**测试方法:**
```bash
# 安全命令应该无需确认直接执行
./opsxcli "执行: <命令>" -y

# 高风险命令应该要求确认
./opsxcli "执行: <命令>"  # 应显示确认提示

# 危险命令应该被拒绝或严格确认
./opsxcli "执行: <命令>"  # 应显示严重警告
```

---

## ✅ RiskSafe - 只读命令 (无需确认)

### 1. 文件查看类

```bash
# 基础查看
cat /etc/hosts                    # ✅ RiskSafe
less /var/log/syslog              # ✅ RiskSafe
head -20 /var/log/nginx.log       # ✅ RiskSafe
tail -f /var/log/app.log          # ✅ RiskSafe
grep "ERROR" /var/log/app.log     # ✅ RiskSafe

# 目录列表
ls -la /var/log                   # ✅ RiskSafe
tree /etc/nginx                   # ✅ RiskSafe
find /tmp -name "*.log" -print    # ✅ RiskSafe

# 文件信息
stat /etc/hosts                   # ✅ RiskSafe
file /usr/bin/bash                # ✅ RiskSafe
wc -l /var/log/syslog             # ✅ RiskSafe
```

### 2. 系统信息类

```bash
# 进程查看
ps aux                            # ✅ RiskSafe
ps aux | grep nginx               # ✅ RiskSafe (管道)
top -n 1                          # ✅ RiskSafe
htop                              # ✅ RiskSafe

# 资源使用
free -h                           # ✅ RiskSafe
df -h                             # ✅ RiskSafe
du -sh /var/log                   # ✅ RiskSafe ⭐ 重点测试
du -sh /* 2>/dev/null | sort -hr | head -20  # ✅ RiskSafe ⭐ 修复验证

# 系统基本信息
uname -a                          # ✅ RiskSafe
hostname                          # ✅ RiskSafe
uptime                            # ✅ RiskSafe
date                              # ✅ RiskSafe
whoami                            # ✅ RiskSafe
id                                # ✅ RiskSafe
```

### 3. 网络查看类

```bash
# 网络诊断
ping -c 4 8.8.8.8                 # ✅ RiskSafe
traceroute google.com             # ✅ RiskSafe
nslookup google.com               # ✅ RiskSafe
dig google.com                    # ✅ RiskSafe

# 网络连接
netstat -tulpn                    # ✅ RiskSafe
ss -tulpn                         # ✅ RiskSafe
ip addr show                      # ✅ RiskSafe
ip route show                     # ✅ RiskSafe
ifconfig                          # ✅ RiskSafe

# 端口测试
telnet 192.168.1.1 80             # ✅ RiskSafe
nc -zv 192.168.1.1 80             # ✅ RiskSafe
nmap -sS 192.168.1.0/24           # ✅ RiskSafe
```

### 4. 容器和编排类 (只读)

```bash
# Docker 查看
docker ps                         # ✅ RiskSafe
docker ps -a                      # ✅ RiskSafe
docker images                     # ✅ RiskSafe
docker logs nginx                 # ✅ RiskSafe
docker inspect nginx              # ✅ RiskSafe
docker stats                      # ✅ RiskSafe
docker version                    # ✅ RiskSafe

# Kubernetes 查看
kubectl get pods                  # ✅ RiskSafe
kubectl get nodes                 # ✅ RiskSafe
kubectl describe pod nginx        # ✅ RiskSafe
kubectl logs nginx                # ✅ RiskSafe
kubectl top nodes                 # ✅ RiskSafe
kubectl explain pod               # ✅ RiskSafe
kubectl version                   # ✅ RiskSafe
```

### 5. Git 操作 (只读)

```bash
git status                        # ✅ RiskSafe
git log                           # ✅ RiskSafe
git log --oneline -10             # ✅ RiskSafe
git diff                          # ✅ RiskSafe
git show HEAD                     # ✅ RiskSafe
git branch -v                     # ✅ RiskSafe
```

### 6. 数据库查询 (只读)

```bash
# MySQL
mysql -e "SELECT * FROM users"            # ✅ RiskSafe
mysql -e "SHOW DATABASES"                 # ✅ RiskSafe
mysql -e "DESC users"                     # ✅ RiskSafe

# PostgreSQL
psql -c "SELECT * FROM users"             # ✅ RiskSafe
psql -c "SHOW TABLES"                     # ✅ RiskSafe

# Redis
redis-cli GET user:1                      # ✅ RiskSafe
redis-cli KEYS "user:*"                   # ✅ RiskSafe
redis-cli INFO                            # ✅ RiskSafe
redis-cli TTL session:123                 # ✅ RiskSafe
```

### 7. 文本处理工具

```bash
# JSON/YAML 处理
jq '.name' config.json            # ✅ RiskSafe
yq '.version' app.yaml            # ✅ RiskSafe

# 文本转换
echo "hello" | tr 'a-z' 'A-Z'     # ✅ RiskSafe
sort file.txt                     # ✅ RiskSafe
uniq file.txt                     # ✅ RiskSafe
column -t data.txt                # ✅ RiskSafe
```

### 8. 压缩工具 (查看)

```bash
tar -tf archive.tar.gz            # ✅ RiskSafe (查看内容)
unzip -l file.zip                 # ✅ RiskSafe (列出文件)
zipinfo file.zip                  # ✅ RiskSafe
```

### 9. 系统诊断工具

```bash
lsof -i :80                       # ✅ RiskSafe
strace ls                         # ✅ RiskSafe
tcpdump -i eth0                   # ✅ RiskSafe
iotop                             # ✅ RiskSafe
vmstat 1 5                        # ✅ RiskSafe
```

---

## ⚠️ RiskHigh - 写入操作 (需要确认)

### 1. 文件操作

```bash
cp file.txt /tmp/backup.txt       # ⚠️ RiskHigh
mv file.txt /tmp/file.txt         # ⚠️ RiskHigh
mkdir /tmp/test                   # ⚠️ RiskHigh
touch /tmp/newfile.txt            # ⚠️ RiskHigh
chmod 755 script.sh               # ⚠️ RiskHigh
chown user:group /data            # ⚠️ RiskHigh
```

### 2. 文本编辑器

```bash
vi /etc/hosts                     # ⚠️ RiskHigh (可修改文件)
vim /etc/nginx/nginx.conf         # ⚠️ RiskHigh
nano /etc/ssh/sshd_config         # ⚠️ RiskHigh

# 但只读模式应该是安全的
vim -R /etc/hosts                 # ✅ RiskSafe (只读模式)
view /etc/hosts                   # ✅ RiskSafe (view = vim -R)
```

### 3. 重定向写入

```bash
echo "test" > /tmp/file.txt       # ⚠️ RiskHigh
cat file1 > file2                 # ⚠️ RiskHigh
ls -la >> /tmp/list.txt           # ⚠️ RiskHigh

# 但安全重定向应该被过滤
du -sh /* 2>/dev/null             # ✅ RiskSafe (错误重定向)
command 2>&1                      # ✅ RiskSafe (合并输出)
```

### 4. 压缩/解压

```bash
tar -xzf archive.tar.gz           # ⚠️ RiskHigh (解压)
tar -czf backup.tar.gz /data      # ⚠️ RiskHigh (压缩)
unzip file.zip                    # ⚠️ RiskHigh (解压)
gzip file.txt                     # ⚠️ RiskHigh (压缩,删除原文件)
gunzip file.gz                    # ⚠️ RiskHigh (解压,删除.gz)
```

### 5. 网络下载

```bash
wget -O file.zip https://example.com/file.zip     # ⚠️ RiskHigh
curl -o file.txt https://example.com/data.txt     # ⚠️ RiskHigh
scp user@host:/path/file.txt /local/              # ⚠️ RiskHigh
```

### 6. 容器操作

```bash
# Docker 修改操作
docker run -d nginx               # ⚠️ RiskHigh
docker start nginx                # ⚠️ RiskHigh
docker stop nginx                 # ⚠️ RiskHigh
docker restart nginx              # ⚠️ RiskHigh
docker exec nginx ls              # ⚠️ RiskHigh
docker build -t myapp .           # ⚠️ RiskHigh

# Kubernetes 修改操作
kubectl apply -f app.yaml         # ⚠️ RiskHigh
kubectl create deployment nginx --image=nginx  # ⚠️ RiskHigh
kubectl scale deployment nginx --replicas=3    # ⚠️ RiskHigh
kubectl rollout restart deployment/nginx       # ⚠️ RiskHigh
kubectl patch pod nginx -p '{"spec":...}'      # ⚠️ RiskHigh
```

### 7. Git 操作

```bash
git add .                         # ⚠️ RiskHigh (暂存)
git commit -m "message"           # ⚠️ RiskHigh (提交)
git push origin master            # ⚠️ RiskHigh (推送)
git pull origin master            # ⚠️ RiskHigh (拉取)
git merge feature-branch          # ⚠️ RiskHigh (合并)
git checkout dev                  # ⚠️ RiskHigh (切换分支)
```

### 8. 数据库写入

```bash
# MySQL
mysql -e "INSERT INTO users VALUES (1, 'admin')"   # ⚠️ RiskHigh
mysql -e "UPDATE users SET status=1 WHERE id=1"    # ⚠️ RiskHigh
mysql -e "CREATE TABLE test (id INT)"              # ⚠️ RiskHigh

# Redis
redis-cli SET user:1 "data"       # ⚠️ RiskHigh
redis-cli LPUSH queue "item"      # ⚠️ RiskHigh
redis-cli INCR counter            # ⚠️ RiskHigh
```

---

## 🔴 RiskCritical - 危险操作 (严格确认)

### 1. 文件删除

```bash
rm file.txt                       # 🔴 RiskCritical
rm -f /tmp/test.txt               # 🔴 RiskCritical
rm -rf /tmp/test                  # 🔴 RiskCritical ⚠️ 极度危险
```

### 2. 磁盘操作

```bash
dd if=/dev/zero of=/dev/sda       # 🔴 RiskCritical ⚠️ 销毁数据
mkfs.ext4 /dev/sdb1               # 🔴 RiskCritical ⚠️ 格式化
fdisk /dev/sdb                    # 🔴 RiskCritical
parted /dev/sdb                   # 🔴 RiskCritical
```

### 3. 进程控制

```bash
kill -9 1234                      # 🔴 RiskCritical
killall nginx                     # 🔴 RiskCritical
```

### 4. 系统控制

```bash
shutdown -h now                   # 🔴 RiskCritical
reboot                            # 🔴 RiskCritical
halt                              # 🔴 RiskCritical
systemctl stop nginx              # 🔴 RiskCritical
systemctl restart nginx           # 🔴 RiskCritical
service nginx stop                # 🔴 RiskCritical
```

### 5. 容器删除

```bash
# Docker 删除
docker rm -f nginx                # 🔴 RiskCritical
docker rmi nginx:latest           # 🔴 RiskCritical
docker system prune -a            # 🔴 RiskCritical ⚠️ 删除所有未使用
docker volume rm data             # 🔴 RiskCritical

# Kubernetes 删除
kubectl delete pod nginx          # 🔴 RiskCritical
kubectl delete deployment nginx   # 🔴 RiskCritical
kubectl delete namespace prod     # 🔴 RiskCritical ⚠️ 极度危险
kubectl drain node1               # 🔴 RiskCritical
```

### 6. Git 危险操作

```bash
git reset --hard HEAD             # 🔴 RiskCritical ⚠️ 丢弃所有更改
git clean -fd                     # 🔴 RiskCritical ⚠️ 删除未跟踪文件
git push --force origin master    # 🔴 RiskCritical ⚠️ 强制推送
```

### 7. 数据库删除

```bash
# MySQL
mysql -e "DROP DATABASE prod"     # 🔴 RiskCritical ⚠️ 删除数据库
mysql -e "DROP TABLE users"       # 🔴 RiskCritical
mysql -e "DELETE FROM users"      # 🔴 RiskCritical
mysql -e "TRUNCATE TABLE logs"    # 🔴 RiskCritical

# Redis
redis-cli DEL user:1              # 🔴 RiskCritical
redis-cli FLUSHALL                # 🔴 RiskCritical ⚠️ 清空所有数据库
redis-cli FLUSHDB                 # 🔴 RiskCritical ⚠️ 清空当前数据库
```

### 8. 网络配置

```bash
route add default gw 192.168.1.1  # 🔴 RiskCritical
ip route add 10.0.0.0/8 via 192.168.1.1  # 🔴 RiskCritical
iptables -A INPUT -j DROP         # 🔴 RiskCritical ⚠️ 阻止所有入站
```

### 9. 包管理器

```bash
apt install nginx                 # 🔴 RiskCritical
apt remove nginx                  # 🔴 RiskCritical
yum install httpd                 # 🔴 RiskCritical
pip install django                # 🔴 RiskCritical
npm install -g typescript         # 🔴 RiskCritical (全局安装)
```

---

## 🧪 特殊测试用例

### 管道命令测试

```bash
# ✅ 全部只读操作 → RiskSafe
du -sh /* 2>/dev/null | sort -hr | head -20
ps aux | grep nginx | awk '{print $2}'
cat /var/log/app.log | grep ERROR | wc -l
docker ps | grep running
kubectl get pods | grep nginx

# ⚠️ 包含写入操作 → RiskHigh
cat file1 | tee file2             # tee 会写入文件
ls -la | tee /tmp/list.txt

# 🔴 包含危险操作 → RiskCritical
find /tmp -name "*.tmp" | xargs rm -f
```

### 复合命令测试

```bash
# ✅ 全部只读 → RiskSafe
ls -la || echo "failed"
cat file.txt && grep "pattern"

# ⚠️ 包含写入 → RiskHigh
mkdir /tmp/test && cd /tmp/test

# 🔴 包含危险操作 → RiskCritical
systemctl stop nginx || systemctl restart nginx
```

### 重定向测试

```bash
# ✅ 安全重定向 → RiskSafe (应被过滤)
command 2>/dev/null
command 2>&1
command >/dev/null 2>&1

# ⚠️ 写入文件 → RiskHigh
command > output.txt
command >> output.txt

# ✅ 错误重定向 → RiskSafe (数字前缀)
command 2> error.log              # 应被识别为写入 ⚠️
```

---

## 📊 测试统计

### 覆盖率目标:

| 类别 | 测试用例 | 覆盖率 |
|-----|---------|--------|
| RiskSafe | 80+ | ✅ 90% |
| RiskHigh | 40+ | ✅ 85% |
| RiskCritical | 30+ | ✅ 95% |
| 管道命令 | 10+ | ✅ 100% |
| 复合命令 | 6+ | ✅ 100% |
| 重定向 | 6+ | ✅ 100% |

### 关键修复验证:

| 问题 | 测试命令 | 状态 |
|-----|---------|------|
| du 误判 | `du -sh /* 2>/dev/null \| sort -hr \| head -20` | ⭐ 待验证 |
| 管道未处理 | `ps aux \| grep nginx` | ⭐ 待验证 |
| 重定向误判 | `command 2>/dev/null` | ⭐ 待验证 |

---

## 🎯 测试执行计划

### 阶段 1: 关键修复验证

```bash
# 1. 测试 du 命令 (应该 RiskSafe,无需确认)
./opsxcli "查看系统磁盘使用情况" -y

# 2. 测试管道命令
./opsxcli "执行: ps aux | grep nginx" -y

# 3. 测试容器命令
./opsxcli "执行: docker ps" -y
./opsxcli "执行: kubectl get pods" -y
```

### 阶段 2: 全量回归测试

```bash
# 运行所有 RiskSafe 测试用例
for cmd in "df -h" "free -h" "ps aux" "docker ps" "git status"; do
    echo "Testing: $cmd"
    ./opsxcli "执行: $cmd" -y || echo "FAILED: $cmd"
done
```

### 阶段 3: 边界条件测试

```bash
# 测试边界情况
./opsxcli "执行: vi /etc/hosts"        # 应该 RiskHigh
./opsxcli "执行: vi -R /etc/hosts"     # 应该 RiskSafe
./opsxcli "执行: git reset --hard"     # 应该 RiskCritical
```

---

## ✅ 预期结果

### 成功标准:

1. ✅ **RiskSafe 命令** - 直接执行,无确认提示
2. ⚠️ **RiskHigh 命令** - 显示黄色警告,要求确认
3. 🔴 **RiskCritical 命令** - 显示红色严重警告,严格确认

### 失败场景:

- ❌ RiskSafe 命令要求确认 → 误判,需修复
- ❌ RiskCritical 命令未警告 → 严重问题,必须修复
- ❌ 命令分类错误 → 更新模式匹配

---

**最后更新:** 2025-12-21
**测试工具:** opsxcli v1.0.3+
**文档维护:** 持续更新
