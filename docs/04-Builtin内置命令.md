# Builtin 内置命令规划

## 📋 目标

将内置基础命令用 Go 原生实现，集成到 opsxcli，使其成为更全面的运维工具箱。

## 🎯 集成策略

### 策略 1: 命令别名/转发（推荐）

**优点**:
- 实现简单，不增加二进制体积
- 利用系统已有命令
- 保持兼容性

**实现方式**:
```go
// opsxcli ls -> 转发到系统 ls 命令
// opsxcli cat file.txt -> 转发到系统 cat 命令
```

### 策略 2: Go 原生实现

**优点**:
- 完全独立，不依赖系统命令
- 可控性强

**缺点**:
- 显著增加体积（每个命令约 0.5-2MB）
- 开发成本高
- 难以达到 C 语言版本的性能

### 策略 3: 混合模式（最佳实践）

- **高频运维命令**: Go 原生实现（如 ss、netstat、top、ifconfig）
- **基础文件命令**: 转发到系统命令（如 ls、cp、mv、cat）

## 📦 Builtin 命令分析

根据 `基础命令清单` 的内容：

### ✅ 已实现 (网络相关)

| Builtin 命令 | opsxcli 对应 | 状态 |
|-------------|-------------|------|
| ping | opsxcli ping | ✅ 已实现 |
| traceroute | opsxcli traceroute | ✅ 已实现 |
| telnet | opsxcli telnet | ✅ 已实现 |
| nc | opsxcli nc | ✅ 已实现 |
| netstat | opsxcli netstat | ✅ 已实现 |
| wget | opsxcli wget | ✅ 已实现 |
| top | opsxcli sys | ✅ 已实现 (TUI增强版) |
| vi | opsxcli vi | ✅ 已实现 (转发) |
| vim | opsxcli vim | ✅ 已实现 (转发) |

### 🔄 可转发的基础命令

这些命令建议直接转发到系统命令，不增加体积：

#### 文件操作
```bash
ls, cp, mv, rm, mkdir, rmdir, touch, chmod, chown, ln
cat, more, less, head, tail, grep, awk, sed
```

#### 归档压缩
```bash
tar, gzip, unzip
```

#### 进程管理
```bash
ps, kill, pstree
```

#### 系统信息
```bash
uname, hostname, whoami, id, free, df, du, mount, umount
```

#### 时间日期
```bash
date, time, sleep, watch, hwclock
```

### 🎯 推荐原生实现的命令

基于运维场景，以下命令值得用 Go 实现增强版本：

| 命令 | 优先级 | 原因 |
|-----|-------|------|
| ifconfig | 高 | 网络配置核心命令 |
| route | 高 | 路由管理 |
| ip | 高 | 现代网络管理 |
| tree | 中 | 目录树可视化 |
| ps | 中 | 进程信息（已在 sys 中部分实现） |

## 🚀 实现计划

### 阶段 1: 命令转发框架（快速实现）

创建通用的命令转发机制：

```go
// cmd/builtin.go
func NewBuiltinCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "builtin [command] [args...]",
        Short: "Builtin内置命令（转发到系统命令）",
        RunE: func(cmd *cobra.Command, args []string) error {
            if len(args) == 0 {
                return fmt.Errorf("请指定要执行的命令")
            }
            // 转发到系统命令
            return runSystemCommand(args[0], args[1:]...)
        },
    }
    return cmd
}
```

**支持方式**:
```bash
opsxcli ls -la
opsxcli cat /etc/hosts
```

### 阶段 2: 直接命令别名（用户友好）

添加常用命令的直接别名：

```bash
opsxcli ls -la          # 等价于 ls -la
opsxcli cat file.txt    # 等价于 cat file.txt
```

**实现**:
```go
// cmd/root.go
rootCmd.AddCommand(
    createAliasCmd("ls", "列出目录内容"),
    createAliasCmd("cat", "显示文件内容"),
    createAliasCmd("grep", "搜索文本"),
    // ... 更多别名
)
```

### 阶段 3: 高级功能增强（可选）

为关键命令提供增强版本：

```bash
opsxcli tree --color      # 彩色目录树
opsxcli ifconfig --json   # JSON格式输出
opsxcli ps --tree         # 进程树视图
```

## 📊 体积影响评估

| 实现方式 | 预计体积增加 | 命令数量 |
|---------|------------|---------|
| 纯转发 | +0.5MB | 30+ |
| 混合模式（5个原生实现） | +3-5MB | 30+ |
| 全部原生实现 | +15-25MB | 30+ |

**推荐**: 混合模式，最终体积约 18-20MB

## 🎨 命令分组优化

更新 `opsxcli --help` 输出：

```
Commands:

  数据库:  MySQL, PostgreSQL, Redis 操作
    mysql, psql, redis

  网络:  SSH, Telnet, Ping, 端口扫描等
    ssh, telnet, nc, ping, traceroute, ss, nmap

  监控:  系统监控(htop风格), 网络监控(iftop风格)
    sys, net

  文件:  文件和目录操作 (builtin 内置)
    ls, cp, mv, rm, cat, grep, tree

  系统:  进程和系统信息
    ps, top, free, df, du

  网络配置:  网络接口管理
    ifconfig, route, ip
```

## 📝 使用示例

```bash
# 网络工具（已实现）
opsxcli ping 8.8.8.8
opsxcli traceroute google.com
opsxcli ss -tunap

# 系统监控（已实现，2秒实时刷新）
opsxcli sys         # 系统监控
opsxcli net         # 网络监控

# 基础命令（已实现 - Go 原生）
opsxcli ls -la
opsxcli cat /etc/hosts
opsxcli grep "error" /var/log/messages
opsxcli cp file1.txt file2.txt
opsxcli mv old.txt new.txt
opsxcli rm -rf /tmp/test

# 网络配置（已实现 - Go 原生）
opsxcli ifconfig
opsxcli route
opsxcli ip addr show

# 编辑工具（系统转发）
opsxcli vi /etc/hosts
opsxcli vim /etc/hosts
```

## ⚠️ 注意事项

1. **命令冲突处理**
   - 如果系统命令不存在，给出友好提示
   - 提供 `--system` 标志强制使用系统命令

2. **权限问题**
   - 某些命令需要 root 权限（如 ifconfig、route）
   - 提供清晰的权限错误提示

3. **跨平台兼容**
   - Windows 系统的命令映射（如 dir -> ls）
   - 命令参数差异处理

4. **体积控制**
   - 只实现高频运维命令的原生版本
   - 低频命令使用转发方式

## 🔄 下一步行动

1. ✅ **已完成**: 二进制优化 (22M → 15M)
2. ✅ **已完成**: 帮助信息优化
3. ✅ **已完成**: 文档更新
4. ✅ **已完成**: 架构重组
   - [x] 创建 plugins/builtin/ 目录
   - [x] 实现命令转发框架
   - [x] 添加常用命令别名
   - [x] 实现 ifconfig/route/ip 原生版本
   - [x] 实现文件操作命令 (ls, cat, cp, mv, rm, mkdir, touch, grep)
   - [x] 更新帮助信息分组
   - [x] 完成代码质量检查（18/18 测试通过）

## 💡 最终目标

打造一个 **18-20MB** 的全功能运维工具箱：
- 保留 opsxcli 特色功能（TUI监控、SSH工具等）
- 兼容 builtin 基础命令
- 体积适中（比全量 builtin 稍大，但功能更强）
- 完全替代传统 Linux 工具链
