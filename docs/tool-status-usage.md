# 工具状态提示使用指南

## 概述

类似 Claude Code 的工具调用状态提示系统,提供优雅的动画和状态反馈。

## 效果展示

执行网络请求时会显示类似以下效果:

```bash
# 运行时
⠹ 🌐 GET example.com (1.2s)

# 成功时
✓ 🌐 Web Request in 1.23s — 200 - 5.2 kB

# 失败时
✗ 🌐 Web Request in 0.45s — 404 Not Found
```

## 基本使用

### 1. 在插件中集成

```go
package yourplugin

import (
    "time"
    "opsxcli/internal/tui"
)

func YourFunction() error {
    // 创建状态提示
    status := tui.NewToolStatus()

    // 开始执行
    status.Start("your_tool")
    startTime := time.Now()

    // 执行你的逻辑
    err := doSomething()

    // 停止并显示结果
    if err != nil {
        status.Stop(false, err.Error())
        return err
    }

    status.Stop(true, "操作成功")
    return nil
}
```

### 2. 使用自定义消息

```go
// 显示更详细的状态信息
status := tui.NewToolStatus()
status.StartWithMessage("kubectl", "获取 Pods - namespace: default")

// ... 执行操作 ...

status.Stop(true, "找到 5 个 Pods")
```

### 3. 简单的一次性提示

```go
// 不需要动画,只显示一次
tui.SimpleToolStatus("redis", "连接到 localhost:6379")

// 显示成功状态
tui.ToolStatusSuccess("docker", 2*time.Second, "镜像拉取完成")

// 显示错误状态
tui.ToolStatusError("mysql", 1*time.Second, "连接超时")
```

## 工具图标映射

系统内置了常用工具的图标映射:

| 工具类型 | 图标 | 示例工具 |
|---------|------|---------|
| 网络工具 | 🌐 | curl, wget, request |
| 系统工具 | ⚙️ | bash, shell |
| 文件工具 | 📄 | cat, ls, grep |
| Kubernetes | ☸️ | kubectl |
| 容器 | 🐳 | docker |
| 数据库 | 🔴 🐬 🐘 | redis, mysql, postgres |
| 代码工具 | 📝 | git_status, git_diff |

完整映射见 `internal/tui/tool_status.go:getToolIcon()`

## 实际应用示例

### 示例 1: HTTP 请求 (request)

已集成在 `plugins/request/request.go` 中:

```bash
# 测试命令
opsxcli request https://api.github.com

# 显示效果
⠹ 🌐 GET api.github.com
✓ 🌐 Web Request in 0.52s — 200 - 1.2 kB
```

### 示例 2: SSH 连接

```go
func SSHConnect(host string) error {
    status := tui.NewToolStatus()
    status.StartWithMessage("ssh", fmt.Sprintf("连接到 %s", host))

    conn, err := connectSSH(host)
    if err != nil {
        status.Stop(false, "连接失败")
        return err
    }

    status.Stop(true, "连接成功")
    return nil
}
```

### 示例 3: Kubernetes 操作

```go
func GetPods(namespace string) error {
    status := tui.NewToolStatus()
    status.StartWithMessage("kubectl",
        fmt.Sprintf("获取 Pods - namespace: %s", namespace))

    pods, err := kubectlGet("pods", namespace)
    if err != nil {
        status.Stop(false, err.Error())
        return err
    }

    msg := fmt.Sprintf("找到 %d 个 Pods", len(pods))
    status.Stop(true, msg)
    return nil
}
```

## 高级特性

### 动态更新状态消息

虽然当前实现主要用于单个操作,但你可以通过创建多个状态对象来实现步骤化提示:

```go
// 步骤 1
status1 := tui.NewToolStatus()
status1.Start("download")
// ... 下载 ...
status1.Stop(true, "下载完成")

// 步骤 2
status2 := tui.NewToolStatus()
status2.Start("extract")
// ... 解压 ...
status2.Stop(true, "解压完成")
```

### 条件性显示

```go
func SendWithStatus(url string, showStatus bool) error {
    var status *tui.ToolStatus

    if showStatus {
        status = tui.NewToolStatus()
        status.Start("request")
    }

    // ... 执行操作 ...

    if showStatus {
        status.Stop(true, "完成")
    }

    return nil
}
```

## 最佳实践

1. **选择合适的工具名称**: 使用有意义的名称,系统会自动查找对应图标
2. **提供有用的消息**: 成功/失败消息应该简洁但信息丰富
3. **总是调用 Stop**: 确保状态提示正确结束,避免动画卡住
4. **错误处理**: 在任何错误分支都要调用 `Stop(false, errMsg)`

## 扩展自定义图标

如果需要添加新的工具图标,编辑 `internal/tui/tool_status.go`:

```go
func (ts *ToolStatus) getToolIcon(toolName string) string {
    iconMap := map[string]string{
        // ... 现有映射 ...

        // 添加你的工具
        "mytool": "🔧",
        "anothertool": "🚀",
    }
    // ...
}
```

## 性能考虑

- 动画帧率: 80ms (每秒约 12 帧)
- Goroutine: 每个状态对象启动一个独立协程
- 资源: 完成后自动清理,无内存泄漏

## 完整示例项目

查看以下文件了解完整实现:

- `/root/opsxcli/internal/tui/tool_status.go` - 核心实现
- `/root/opsxcli/plugins/request/request.go` - 实际应用
- `/root/opsxcli/internal/tui/screen.go` - 相关 TUI 组件
