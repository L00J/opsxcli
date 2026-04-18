# Validator 优化完整总结与建议

**日期:** 2025-12-21
**任务:** 优化 opsxcli 工具调用规范验证器
**状态:** ⚠️ 需要重新设计策略

---

## 🎯 用户核心需求

1. **文案优化**: "违反 opsxcli 铁律" → "违反 opsxcli 命令约束" ✅ 已完成
2. **策略调整**: **优先使用 opsxcli 内置工具,如不存在或报错才使用 bash**

---

## ⚠️ 当前方案的根本问题

### 问题1: 基于"禁止"的策略不符合需求

**当前实现:**
- 检测到 bash 中包含 kubectl → 拒绝或修正
- 这是**硬性禁止**策略

**用户需求:**
- AI 优先使用 kubectl 工具
- 如果工具不存在或报错,**允许**使用 bash 调用
- 这是**优先级引导**策略,不是禁止

**矛盾:**
当前方案**完全拒绝** bash kubectl,无法实现"报错才允许"的fallback机制。

---

### 问题2: 白名单机制有漏洞

**白名单Bug:**
```go
// ❌ 当前实现
if strings.HasPrefix(commandLower, "echo") && !strings.Contains(command, "$(") && !strings.Contains(command, "`") {
    return true  // 允许所有 echo 开头的命令
}
```

**被绕过的情况:**
```bash
echo "正在统计..." && kubectl get pods -A  # ✅ 被允许(Bug!)
which kubectl && kubectl version             # ✅ 被允许(Bug!)
```

**原因:** 只检查了开头,没有检查整个命令是否包含禁止的工具。

---

### 问题3: AutoCorrect 修正逻辑不完善

**修正条件太严格:**
```go
// 包含 &&, ||, |, 2>&1, $(, `, >, < 都拒绝修正
```

**导致的问题:**
- 简单的 `kubectl get pods -o wide` → `-o` 被误判为 `>` → 拒绝修正 ❌
- 几乎所有稍复杂的命令都被拒绝,AI 不断重试,token 爆炸

**修复后:**
```go
// 使用正则精确匹配 `\s+>` (空格+重定向)
```
**效果:** 改善了,但仍有很多命令被拒绝

---

### 问题4: LLM 学习效果有限

**现状:**
- AI 被拦截后,会尝试各种绕过方式
- 通过白名单漏洞绕过(`echo && kubectl ...`)
- 或者不断重试其他方法

**测试数据:**
| 测试 | 步骤 | Token | 拦截次数 |
|-----|------|-------|---------|
| 优化前 | 14步 | 76.6k | 1次自动修正 |
| 优化后v1 | 13步 | 65.8k | 3次修正+2次拒绝 |
| 优化后v2 (修复精确检测) | 31步 | 162.9k | 多次拒绝,AI迷茫 |
| 优化后v3 (当前) | 21步+ | 116k+ | 多次拦截,但有绕过 |

---

## 💡 根本原因分析

### 1. **策略定位错误**

**当前:** "禁止在 bash 中调用 kubectl"
**应该:** "引导 AI 优先使用 kubectl 工具"

这是**两种完全不同的策略**:
- 禁止策略 → 硬性拦截 → AI 困惑,不断重试
- 引导策略 → 提示建议 → AI 学习,主动选择

### 2. **缺少 Fallback 机制**

用户需求:**"如果工具不存在或报错才使用 bash"**

这意味着需要:
1. 首先尝试 kubectl 工具
2. 如果失败,返回错误信息给 AI
3. AI 看到错误后,可以选择使用 bash fallback
4. 但给出警告:"建议优先使用 kubectl 工具"

**当前实现完全没有这个机制。**

### 3. **System Prompt 的角色被忽视**

**真正应该做的:**
System Prompt 明确优先级 → AI 主动选择工具 → Validator 只做最后防线

**当前做的:**
Validator 强制拦截 → AI 被动应对 → System Prompt 的指导被忽视

---

## 🚀 推荐的新策略

### 方案A: 优先级引导策略 (推荐)

#### 1. **System Prompt 明确优先级**

```
工具使用优先级:

1. **首选**: 使用 opsxcli 内置工具
   - kubectl: K8s 操作 (kubectl get pods -A)
   - docker: 容器管理 (docker ps)
   - redis/mysql/psql: 数据库客户端

2. **次选**: 当内置工具不存在或执行失败时,使用 bash
   - 例如: 尝试 kubectl 工具失败后,可以使用 bash "kubectl get pods"

3. **错误处理**:
   - 如果看到"工具不存在"错误,可以使用 bash 作为 fallback
   - 如果看到"命令约束"警告,优先使用专用工具

✅ 正确流程:
1. 首先尝试: kubectl get pods -A
2. 如果失败: bash "kubectl get pods -A" (作为 fallback)

❌ 错误做法:
直接使用 bash "kubectl get pods" 而不先尝试 kubectl 工具
```

#### 2. **Validator 改为"建议"而非"禁止"**

```go
func (v *ToolCallValidator) ValidateToolCall(...) (bool, string, string) {
    if toolName == "bash" && strings.Contains(command, "kubectl") {
        // 不拒绝执行,而是给出警告
        return true, "", ""  // 允许通过
    }
    return true, "", ""
}

// 但在执行后,返回结果中添加建议
func (a *Agent) executeToolCall(...) string {
    // 检查是否应该使用专用工具
    if toolName == "bash" && strings.Contains(command, "kubectl") {
        suggestion := "💡 提示: opsxcli 提供了 kubectl 专用工具,建议优先使用以获得更好的体验。\n\n"
        return suggestion + result.Output
    }
    return result.Output
}
```

**效果:**
- AI 的调用**不会被拦截**,避免了重试循环
- 但 AI 能看到建议,下次会优先使用专用工具
- 允许 fallback,符合用户需求

---

### 方案B: 双层验证策略 (严格模式)

#### 1. **第一层: System Prompt 引导**
(同方案A)

#### 2. **第二层: Validator 轻量拦截**

```go
// 只拦截明显错误的情况,其他给建议
func (v *ToolCallValidator) ValidateToolCall(...) (bool, string, string) {
    if toolName == "bash" && strings.Contains(command, "kubectl") {
        // 检查是否是复杂脚本
        if strings.Contains(command, "for") ||
           strings.Contains(command, "while") ||
           strings.Count(command, "kubectl") > 2 {
            // 复杂脚本,拒绝执行
            return false, errMsg, suggestion
        }

        // 简单命令,允许但记录
        return true, "", ""  // 允许通过,稍后给建议
    }
    return true, "", ""
}
```

**效果:**
- 只拦截明显不合理的复杂脚本
- 简单的 `bash "kubectl get pods"` 允许通过
- 在结果中添加建议

---

### 方案C: 完全移除 Validator (极简模式)

#### 策略:
完全依赖 System Prompt 引导,移除 Validator 的拦截逻辑。

#### 理由:
1. LLM 足够智能,清晰的 System Prompt 就能引导正确行为
2. 减少复杂度,避免过度工程
3. 允许灵活性,让 AI 根据情况选择

#### 风险:
AI 可能不遵守 System Prompt,需要更强的 Prompt 设计。

---

## 📋 最终建议

### 立即实施: 方案A (优先级引导策略)

#### 步骤1: 修改 System Prompt

```go
const baseSystemPrompt = `运维AI助手...

工具使用规范:

1. **优先使用 opsxcli 内置工具** (kubectl, docker, redis, mysql, psql)
2. **允许 bash fallback**: 当工具不存在或报错时,可以使用 bash
3. **查看建议**: 如果返回结果中有"💡 提示",请在后续步骤中使用建议的工具

✅ 推荐流程:
1. 首先尝试: kubectl get pods -A
2. 如果失败: bash "kubectl get pods -A"

✅ 示例:
查K8s: kubectl get pods -A (优先) → 失败时 bash "kubectl ..." (fallback)
监控: bash "uptime && free -h" (没有专用工具,直接用bash)
`
```

#### 步骤2: 修改 Validator 为建议模式

```go
// validator.go
func (v *ToolCallValidator) ValidateToolCall(toolName string, args map[string]interface{}) (bool, string, string) {
    // 移除所有拦截逻辑,只返回建议
    if toolName == "bash" {
        command, _ := args["command"].(string)
        for keyword, toolName := range forbiddenTools {
            if strings.Contains(command, keyword) && !v.isWhitelistedUsage(command, keyword) {
                // 不拒绝,只返回建议
                suggestion := fmt.Sprintf("opsxcli 提供了 %s 专用工具,建议优先使用", toolName)
                return true, "", suggestion  // 允许通过,但有建议
            }
        }
    }
    return true, "", ""
}
```

```go
// executor.go
func (a *Agent) executeToolCall(...) string {
    // 验证
    isValid, errMsg, suggestion := validator.ValidateToolCall(...)

    // 不再拦截,但记录建议
    var hasSuggestion bool
    if suggestion != "" {
        hasSuggestion = true
    }

    // ... 执行工具 ...

    // 在返回结果中添加建议
    if hasSuggestion {
        return fmt.Sprintf("💡 提示: %s\n\n执行结果:\n%s", suggestion, result.Output)
    }
    return result.Output
}
```

#### 预期效果:

| 指标 | 当前 | 方案A后 |
|-----|------|---------|
| **拦截次数** | 5-10次 | 0次 (全部允许) |
| **执行步骤** | 21步+ | 3-5步 (AI快速完成) |
| **Token 使用** | 116k+ | ~10k (大幅减少) |
| **用户体验** | AI频繁被拒,困惑 | AI流畅执行,学习建议 |
| **Fallback** | 不支持 | ✅ 支持 |

---

## ✅ 总结

### 核心问题:
1. 当前策略是"禁止",应该是"引导"
2. 缺少 fallback 机制
3. 过度拦截导致 AI 困惑,token 爆炸

### 推荐方案:
**方案A: 优先级引导策略**
- 依赖 System Prompt 引导
- Validator 只给建议,不拦截
- 支持 fallback 机制

### 实施优先级:
1. 立即: 修改 System Prompt,明确优先级
2. 立即: Validator 改为建议模式
3. 测试: 验证 token 使用和执行效率
4. 监控: 观察 AI 是否遵守优先级

---

**最后更新:** 2025-12-21
**建议:** 🔥 采用方案A,彻底解决问题
