# OpsX CLI 工具实现方式说明

## 🎯 两种实现方式

OpsX CLI 中的工具采用两种实现方式,各有优劣:

### 1. **纯 Go 实现** ✅ 推荐

**特点:**
- 跨平台兼容性好
- 不依赖系统命令
- 性能更好
- 可控性强
- 可以集成状态提示

**示例:**

#### curl (纯 Go 实现)
```go
// plugins/curl/curl.go
package curl

import (
    "net/http"
    "opsxcli/internal/tui"  // 可以集成状态提示
)

func Request(url string) error {
    // 创建状态提示
    status := tui.NewToolStatus()
    status.Start("curl")

    // 使用 Go 标准库
    req, err := http.NewRequest("GET", url, nil)
    resp, err := http.DefaultClient.Do(req)

    // 停止状态提示
    status.Stop(err == nil, "完成")
    return err
}
```

**已实现的纯 Go 工具:**

| 工具 | 实现方式 | 文件位置 | 状态提示 |
|-----|---------|---------|----------|
| curl | Go `net/http` | `plugins/curl/curl.go` | ❌ 待集成 |
| wget | Go `net/http` | `plugins/wget/wget.go` | ❌ 待集成 |
| request | Go `net/http` | `plugins/request/request.go` | ✅ 已集成 |
| websearch | Go `net/http` | `plugins/websearch/websearch.go` | ✅ 已集成 |
| mysql | MySQL Go Driver | `plugins/mysql/` | ❌ 待集成 |
| redis | Redis Go Client | `plugins/redis/` | ❌ 待集成 |
| psql | PostgreSQL Go Driver | `plugins/postgres/` | ❌ 待集成 |
| docker | Docker Go SDK | `plugins/docker/` | ❌ 待集成 |
| kubectl | Kubernetes Go Client | `plugins/kubernetes/` | ❌ 待集成 |
| ssh | Go SSH Library | `plugins/ssh/` | ❌ 待集成 |
| telnet | Go Net Library | `plugins/telnet/` | ❌ 待集成 |

### 2. **Bash 命令调用** ⚠️ 有限制

**特点:**
- 依赖系统命令存在
- 跨平台兼容性差
- 难以集成状态提示
- 适合简单封装

**示例:**

#### 通过 tools.CommandTool (Bash 调用)
```go
// internal/tools/builtin.go
func NewCatTool() *CommandTool {
    return NewCommandTool(
        "cat",
        "查看文件内容",
        parameters,
        RiskSafe,
        func(args map[string]interface{}) (string, []string, error) {
            file := args["file"].(string)
            // 返回: 命令名, 参数列表
            return "opsxcli", []string{"cat", file}, nil
        },
    )
}

// tool.go:97
cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
output, err := cmd.CombinedOutput()
```

**通过 Bash 调用的工具:**

| 工具类型 | 工具 | 调用方式 | 限制 |
|---------|-----|---------|------|
| 文件操作 | cat, ls, grep, head, tail | exec.Command | ⚠️ 依赖系统命令 |
| 进程管理 | ps, top, kill | exec.Command | ⚠️ 依赖系统命令 |
| 系统信息 | free, df, du, uname | exec.Command | ⚠️ 依赖系统命令 |
| 文件管理 | cp, mv, rm, mkdir | exec.Command | ⚠️ 依赖系统命令 |
| 权限管理 | chmod, chown | exec.Command | ⚠️ 依赖系统命令 |
| Git 操作 | git status, git diff | exec.Command | ⚠️ 依赖 git |
| 智能 Bash | smart_bash | exec.Command("bash", "-c") | ⚠️ 危险 |

## 🔍 如何区分?

### 检查实现方式:

```bash
# 1. 查看是否有独立的 plugin 目录
ls plugins/curl/       # 有 -> 纯 Go 实现
ls plugins/websearch/  # 有 -> 纯 Go 实现

# 2. 查看 internal/tools/builtin.go
grep "NewCatTool" internal/tools/builtin.go  # Bash 调用
grep "NewWebSearchTool" internal/tools/builtin.go  # 纯 Go

# 3. 检查代码
grep -r "exec.Command" plugins/curl/     # 没有 -> 纯 Go
grep -r "net/http" plugins/curl/         # 有 -> 纯 Go
```

## 📊 对比表

| 特性 | 纯 Go 实现 | Bash 调用 |
|-----|----------|-----------|
| **跨平台** | ✅ 完美支持 | ❌ Windows 受限 |
| **依赖** | ✅ 无外部依赖 | ❌ 需要系统命令 |
| **性能** | ✅ 高性能 | ⚠️ 进程创建开销 |
| **状态提示** | ✅ 易于集成 | ❌ 难以集成 |
| **错误处理** | ✅ 精确控制 | ⚠️ 受限于命令输出 |
| **定制化** | ✅ 高度可控 | ❌ 受限于命令参数 |
| **开发成本** | ⚠️ 较高 | ✅ 低 |

## 🚀 状态提示集成示例

### ✅ 纯 Go 实现 (易于集成)

```go
// plugins/curl/curl.go
package curl

import (
    "net/http"
    "opsxcli/internal/tui"
)

func Request(url string) error {
    // 创建状态提示
    status := tui.NewToolStatus()
    statusMsg := fmt.Sprintf("GET %s", url)
    status.StartWithMessage("curl", statusMsg)

    // 执行请求
    resp, err := http.Get(url)

    // 停止状态提示
    if err != nil {
        status.Stop(false, err.Error())
        return err
    }

    msg := fmt.Sprintf("%d - %s", resp.StatusCode, formatSize(resp.ContentLength))
    status.Stop(true, msg)
    return nil
}
```

**效果:**
```
⠹ 🌐 GET example.com
✓ 🌐 curl in 0.52s — 200 - 1.2 kB
```

### ❌ Bash 调用 (难以集成)

```go
// internal/tools/builtin.go
func NewCatTool() *CommandTool {
    return NewCommandTool(
        "cat",
        "查看文件内容",
        parameters,
        RiskSafe,
        func(args map[string]interface{}) (string, []string, error) {
            // ❌ 这里无法添加状态提示
            // ❌ CommandTool.Execute 中使用 exec.Command
            // ❌ 状态提示需要在 Execute 外层控制
            return "cat", []string{file}, nil
        },
    )
}
```

**问题:**
- CommandTool 的 Execute 方法在 `tool.go:86` 中统一处理
- 无法为单个工具定制状态提示
- 需要修改 CommandTool 基类才能支持

## 💡 推荐做法

### 1. 新工具优先使用纯 Go 实现

```go
// ✅ 推荐
plugins/
├── mytool/
│   └── mytool.go   // 纯 Go 实现,可集成状态提示
```

### 2. 为纯 Go 工具集成状态提示

```go
// ✅ 推荐
func MyFunction() error {
    status := tui.NewToolStatus()
    status.Start("mytool")

    // ... 执行逻辑 ...

    status.Stop(true, "完成")
    return nil
}
```

### 3. Bash 工具优化方案

如果必须使用 Bash 调用,可以:

#### 方案 A: 创建 Wrapper 函数

```go
// cmd/cat.go
func RunCat(file string) error {
    status := tui.NewToolStatus()
    status.Start("cat")

    // 调用系统命令
    cmd := exec.Command("cat", file)
    output, err := cmd.Output()

    if err != nil {
        status.Stop(false, err.Error())
        return err
    }

    status.Stop(true, fmt.Sprintf("%d bytes", len(output)))
    fmt.Print(string(output))
    return nil
}
```

#### 方案 B: 修改 CommandTool 基类

```go
// internal/tools/tool.go
func (t *CommandTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
    // 创建状态提示
    status := tui.NewToolStatus()
    status.Start(t.name)

    // 构建并执行命令
    cmdName, cmdArgs, err := t.buildCmd(args)
    cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
    output, err := cmd.CombinedOutput()

    // 停止状态提示
    success := err == nil
    status.Stop(success, string(output))

    return &ToolResult{
        Success: success,
        Output:  string(output),
    }, nil
}
```

## 📝 总结

### curl 和 wget 的实现方式

**答案:** curl 和 wget 在 opsxcli 中是 **纯 Go 实现**,**不是** Bash 命令调用!

| 工具 | 实现方式 | 文件 | 核心代码 |
|-----|---------|------|---------|
| curl | ✅ 纯 Go | `plugins/curl/curl.go` | `http.NewRequest()` |
| wget | ✅ 纯 Go | `plugins/wget/wget.go` | `http.Get()` |
| request | ✅ 纯 Go | `plugins/request/request.go` | `http.NewRequest()` |
| websearch | ✅ 纯 Go | `plugins/websearch/websearch.go` | `http.NewRequest()` |

### 其他 Bash 工具

以下工具通过 `exec.Command` 调用系统命令:

```go
// internal/tools/builtin.go
cat, ls, grep, head, tail, tree,      // 文件工具
ps, top, free, df, du, uname,         // 系统工具
mkdir, chmod, chown, touch,           // 文件管理
ping, ss, netstat, ifconfig,          // 网络工具 (部分)
git_status, git_diff,                 // Git 工具
bash (smart_bash)                      // 智能 Bash
```

### 迁移建议

如果要为 curl/wget 添加状态提示,直接修改它们的 Go 实现即可:

```bash
# 1. 编辑文件
vim plugins/curl/curl.go

# 2. 添加状态提示 (参考 plugins/request/request.go)
import "opsxcli/internal/tui"

# 3. 重新编译
go build -o opsxcli
```

这样就能获得和 `request` 一样的优雅状态提示效果!
