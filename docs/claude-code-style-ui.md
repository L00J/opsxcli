# 类 Claude Code 工具状态提示系统

本文档介绍 OpsX CLI 中集成的类 Claude Code 风格的工具调用状态提示系统。

## 🎨 功能特性

### 1. 动态状态提示
- ✨ 流畅的动画效果 (80ms 帧率)
- 🎯 自动识别工具类型并显示对应图标
- ⏱️ 实时显示执行时长
- ✅/❌ 成功/失败状态反馈
- 📊 智能显示执行结果(大小、状态码等)

### 2. Web 搜索工具
- 🌐 支持多个搜索引擎(国内外)
- 🔓 无需 API Token
- 🖥️ 模拟浏览器访问
- 📝 可选保存 HTML 结果
- 🚀 并发搜索多个引擎

## 📦 已集成的功能

### HTTP 请求工具 (request)

**位置**: `plugins/request/request.go`

**特性**:
- 显示请求方法和目标域名
- 实时动画反馈
- 自动计算响应大小
- 状态码验证

**使用示例**:
```bash
# 基本请求
opsxcli request https://api.github.com

# 效果展示
⠹ 🌐 GET api.github.com          # 执行中
✓ 🌐 Web Request in 0.52s — 200 - 1.2 kB  # 成功

# POST 请求
opsxcli request https://api.example.com \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}'
```

### Web 搜索工具 (websearch)

**位置**: `plugins/websearch/websearch.go`

**支持的搜索引擎**:

#### 🇨🇳 国内搜索引擎
| 引擎 | 代码 | Token | 说明 |
|-----|------|-------|------|
| 百度 | `baidu` | ✅ 免Token | 最流行的中文搜索 |
| 搜狗 | `sogou` | ✅ 免Token | 搜狗搜索 |
| 360搜索 | `so360` | ✅ 免Token | 360搜索引擎 |
| 必应中国 | `bing_cn` | ✅ 免Token | 微软 Bing 中国版 |

#### 🌍 海外搜索引擎
| 引擎 | 代码 | Token | 说明 |
|-----|------|-------|------|
| Google | `google` | ✅ 免Token | 谷歌搜索 |
| Bing | `bing` | ✅ 免Token | 微软 Bing 国际版 |
| DuckDuckGo | `duckduckgo` | ✅ 免Token | 隐私优先搜索引擎 |
| Yahoo | `yahoo` | ✅ 免Token | 雅虎搜索 |

**使用示例**:
```bash
# 列出所有搜索引擎
opsxcli websearch --list

# 默认搜索 (百度 + Google)
opsxcli websearch "语雀 API 文档"

# 指定搜索引擎
opsxcli websearch "Claude Code" --engines google,bing,baidu

# 国内专用搜索
opsxcli websearch "Python 教程" --engines baidu,sogou,so360

# 保存 HTML 结果
opsxcli websearch "Kubernetes" --engines google --save-html

# 效果展示
⠹ 🔍 Web Search "语雀 API 文档" via 百度, Google
✓ 🔍 Web Search in 1.23s — 找到 2 个搜索结果

搜索结果:
────────────────────────────────────────────────────────────────────────────────
🇨🇳 百度
  URL: https://www.baidu.com/s?wd=%E8%AF%AD%E9%9B%80+API...
  大小: 125.3 kB | 耗时: 523ms

🌍 Google
  URL: https://www.google.com/search?q=%E8%AF%AD%E9%9B%80+API...
  大小: 87.2 kB | 耗时: 789ms
```

## 🎯 工具图标映射

系统内置了丰富的工具图标:

| 类型 | 图标 | 工具 |
|------|------|------|
| 网络工具 | 🌐 | curl, wget, request |
| Web搜索 | 🔍 | websearch |
| 系统工具 | ⚙️ | bash, shell |
| 文件工具 | 📄 | cat, ls, grep |
| Kubernetes | ☸️ | kubectl |
| 容器 | 🐳 | docker |
| 数据库 | 🔴 🐬 🐘 | redis, mysql, postgres |
| SSH | 🔐 | ssh |
| 监控 | 📊 | sys_monitor, net_monitor |

## 💻 开发者指南

### 在你的插件中集成状态提示

```go
package yourplugin

import (
    "time"
    "opsxcli/internal/tui"
)

func YourFunction() error {
    // 1. 创建状态提示
    status := tui.NewToolStatus()

    // 2. 开始执行 (带自定义消息)
    status.StartWithMessage("your_tool", "正在处理...")

    // 3. 执行你的逻辑
    err := doSomething()

    // 4. 停止并显示结果
    if err != nil {
        status.Stop(false, err.Error())
        return err
    }

    status.Stop(true, "操作成功")
    return nil
}
```

### API 参考

#### ToolStatus 方法

```go
// 创建实例
status := tui.NewToolStatus()

// 开始工具调用 (使用默认消息)
status.Start("tool_name")

// 开始工具调用 (自定义消息)
status.StartWithMessage("tool_name", "自定义状态消息")

// 停止并显示结果
status.Stop(success bool, message string)

// 简单的一次性提示 (无动画)
tui.SimpleToolStatus("tool_name", "消息")
tui.ToolStatusSuccess("tool_name", duration, "成功消息")
tui.ToolStatusError("tool_name", duration, "错误消息")
```

#### 添加自定义图标

编辑 `internal/tui/tool_status.go`:

```go
func (ts *ToolStatus) getToolIcon(toolName string) string {
    iconMap := map[string]string{
        // ... 现有映射 ...
        "mytool": "🔧",
        "newtool": "🚀",
    }
    // ...
}
```

## 🎨 效果展示

### HTTP 请求
```
⠹ 🌐 GET api.github.com
✓ 🌐 Web Request in 0.52s — 200 - 1.2 kB
```

### Web 搜索
```
⠹ 🔍 Web Search "语雀 API" via 百度, Google
✓ 🔍 Web Search in 1.23s — 找到 2 个搜索结果
```

### SSH 连接
```
⠹ 🔐 SSH 连接到 user@host
✓ 🔐 SSH in 1.45s — 连接成功
```

### Kubernetes 操作
```
⠹ ☸️ 获取 Pods - namespace: default
✓ ☸️ Kubernetes in 2.31s — 找到 5 个 Pods
```

## 🔧 实现细节

### 文件结构
```
internal/tui/
├── tool_status.go      # 工具状态提示核心实现
├── screen.go           # TUI 屏幕管理
└── ...

plugins/
├── request/
│   └── request.go      # HTTP 请求 (已集成)
├── websearch/
│   └── websearch.go    # Web 搜索 (已集成)
└── ...

cmd/
├── request.go          # HTTP 请求命令
├── websearch.go        # Web 搜索命令
└── ...
```

### 性能特性
- **动画帧率**: 80ms (约 12 FPS)
- **协程管理**: 每个状态对象一个独立协程
- **资源清理**: 自动清理,无内存泄漏
- **并发安全**: 使用 RWMutex 保护共享状态

## 📝 最佳实践

1. **选择合适的工具名称**: 使用预定义的工具名称可获得对应图标
2. **提供有意义的消息**: 成功/失败消息应简洁但信息丰富
3. **总是调用 Stop**: 确保状态提示正确结束
4. **错误处理**: 在任何错误分支都要调用 `Stop(false, errMsg)`
5. **条件性显示**: 提供 `showStatus` 参数让用户选择是否显示状态

## 🚀 未来计划

- [ ] 支持多步骤进度显示
- [ ] 支持嵌套工具调用
- [ ] 添加更多搜索引擎 (知乎、CSDN 等)
- [ ] 智能搜索结果解析和摘要
- [ ] Web 搜索结果缓存
- [ ] 支持自定义动画样式

## 📚 相关文档

- [工具状态使用指南](./tool-status-usage.md)
- [OpsX CLI 开发规则](../.claude/CLAUDE.md)
- [内部工具开发](../internal/tools/README.md)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request!

## 📄 许可证

MIT License
