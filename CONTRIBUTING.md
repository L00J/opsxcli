# CONTRIBUTING.md — opsxcli 贡献者指南

感谢你对 opsxcli 的兴趣！本文档帮助你快速开始贡献。

---

## 快速开始

```bash
# 1. Fork 并克隆仓库
git clone https://github.com/YOUR_FORK/opsxcli.git
cd opsxcli

# 2. 安装依赖（Go 1.24.2+）
go mod download

# 3. 验证环境
make build
make test
```

## 开发工作流

```bash
# 创建功能分支
git checkout -b feature/your-feature-name

# 开发并测试
make build        # 构建验证
make test         # 运行测试
make fmt          # 格式化
make lint         # 代码检查（需要 golangci-lint）

# 提交前检查（必须全部通过）
make fmt && make test && make lint

# 提交并推送
git add .
git commit -m "feat: 描述你的改动"
git push origin feature/your-feature-name
```

## 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>(<scope>): <描述>

[可选的详细说明]

[可选的 Footer]
```

**类型说明**:

| 类型 | 用途 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 仅文档变更 |
| `style` | 不影响代码含义的变更（格式化） |
| `refactor` | 既不修复 bug 也不添加功能的代码重构 |
| `perf` | 性能优化 |
| `test` | 添加或修复测试 |
| `chore` | 构建过程或辅助工具的变更 |

**示例**:

```
feat(mysql): 添加 SSL 连接支持

fix(agent): 修复安全审批在并发下的竞态条件
docs(ssh): 补充端口转发使用示例
test(redis): 添加集群模式单元测试
```

## 代码规范

### 命名
- 导出函数：`PascalCase`
- 内部函数：`camelCase`
- 所有注释和错误信息：**中文**
- 包名：全小写，不含下划线

### 错误处理
```go
// ✅ 使用 %w 包装错误
if err := db.Connect(); err != nil {
    return fmt.Errorf("连接数据库失败: %w", err)
}

// ✅ 定义 Sentinel 错误
var ErrConnectionFailed = errors.New("连接失败")

// ❌ 不要吞掉错误
_ = doSomething()  // 错误！
```

### 测试要求
- 每个导出函数必须有测试
- 使用表驱动测试
- 测试名称使用中文描述场景

```go
func TestNewClient(t *testing.T) {
    tests := []struct {
        name    string
        host    string
        wantErr bool
    }{
        {name: "正常创建", host: "localhost", wantErr: false},
        {name: "空主机地址", host: "", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := NewClient(tt.host)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewClient() error = %v", err)
            }
        })
    }
}
```

## 添加新插件

详见 `docs/references/plugin-development.md`，简要步骤：

1. `mkdir plugins/<name>/`
2. 实现核心逻辑 + 测试
3. 在 `cmd/<name>.go` 添加命令入口
4. 在 `cmd/root.go` 注册
5. 在 `docs/references/commands/<name>.md` 添加文档

## 添加新 LLM Provider

1. 在 `internal/llm/` 添加 `<provider>.go`
2. 实现 `LLMProvider` 接口
3. 在 `factory.go` 注册
4. 添加测试

## 文档变更

- 修改代码时必须同步更新相关文档
- 新功能必须包含使用文档
- 文档使用 Markdown 格式，中文撰写

## 问题反馈

- **Bug 报告**: 使用 GitHub Issues，提供复现步骤、环境信息（`opsxcli --version`）
- **功能请求**: 描述使用场景和期望行为
- **安全问题**: 请通过邮件私下报告，不要在公开 Issue 中披露

## 社区准则

- 尊重所有参与者
- 使用中文交流
- 讨论技术而非个人
- 帮助他人也是贡献

## 获取帮助

- 阅读 `AGENTS.md` 了解项目架构
- 阅读 `ARCHITECTURE.md` 了解技术细节
- 查阅 `docs/references/` 中的开发文档
- 在 Issue 中提问（使用 `question` 标签）
