# Validator 测试结果分析

**测试时间:** 2025-12-21
**测试版本:** OpsX CLI v1.0.3+ (Validator Enforced)

---

## 📊 测试概览

| 测试 | 任务 | Token | 步骤 | 时间 | Validator触发 |
|-----|------|-------|------|------|--------------|
| **Test 1** | kubectl get po -A | 76.6k | 14步 | ~110s | ✅ 1次修正 |
| **Test 2** | 查询uptime负载 | 3.9k | 1步 | ~4s | ❌ 未触发 |
| **Test 3** | 磁盘读写性能 | 31.8k | 7步 | ~51s | ❌ 未触发 |

---

## ✅ 成功验证: Validator 工作了!

### 测试1: kubectl 工具调用

**关键发现:**
Validator **成功捕获并自动修正**了违规调用!

```
[DEBUG] Tool: bash
[DEBUG] Args: {"command": "kubectl get pods -A --no-headers | wc -l"}
  ⎿ 检测到不合规调用,已自动修正
  ⎿ 原始: bash map[command:kubectl get pods -A --no-headers | wc -l]
  ⎿ 修正为: kubectl map[subcommand:get pods -A --no-headers | wc -l]
```

**证明:**
1. ✅ Validator 正确识别了 "bash kubectl" 违规
2. ✅ 自动修正机制生效 (bash → kubectl 工具)
3. ✅ 显示了清晰的警告信息
4. ✅ 记录了修正统计

---

## ⚠️ 发现的问题

### 问题1: LLM 没有学习到修正 (Critical)

**表现:**
虽然第一次违规被修正,但 AI 在后续步骤中**仍然多次**使用 bash 调用 kubectl:

```bash
Step 9:  bash: "kubectl get pods -A --no-headers | wc -l"  → 被修正 ✅
Step 10: bash: "for ns in $(kubectl get namespaces ...); do kubectl get pods ..." ❌
Step 11: bash: "kubectl get pods -A --no-headers | head -20" ❌
Step 12: bash: "for ns in ...; kubectl get pods -n $ns ..." ❌
Step 13: bash: "kubectl get pods -A --no-headers | wc -l" ❌
Step 14: bash: "kubectl get pods -A --no-headers | awk '{print $1}'" ❌
```

**根本原因:**
- 修正发生在**执行层** (`executor.go`)
- LLM 看到的是**修正后的成功结果**
- LLM **不知道**自己犯了错误
- 下一步推理时,LLM 仍然使用错误的模式

**影响:**
- Validator 虽然保护了系统,但没有改变 LLM 行为
- 每次违规都会触发修正,增加延迟
- Token 统计中会记录大量自动修正

---

### 问题2: bash blacklist 不够全面

**当前 blacklist:**
```go
regexp.MustCompile(`^\s*kubectl\s+`),  // 只匹配行首的 kubectl
regexp.MustCompile(`\|\s*kubectl\s+`), // 只匹配管道后的 kubectl
```

**未覆盖的场景:**
```bash
# ❌ 嵌入在 shell 脚本中的 kubectl (未被捕获)
for ns in $(kubectl get namespaces -o jsonpath='{.items[*].metadata.name}'); do
    kubectl get pods -n $ns
done

# ❌ 命令替换中的 kubectl
echo "总Pod数: $(kubectl get pods -A --no-headers | wc -l)"

# ❌ 条件判断中的 kubectl
if kubectl get pod my-pod 2>/dev/null; then
    echo "Pod exists"
fi
```

**建议:**
扩展 blacklist,或者采用更简单的规则:**只要 bash 命令中包含 "kubectl"/"docker" 关键字,就警告**。

---

### 问题3: Token 使用仍然过高

| 测试 | Token | 期望 | 差距 |
|-----|-------|------|------|
| kubectl | 76.6k | ~10k | **+660%** |
| 磁盘性能 | 31.8k | ~8k | **+297%** |

**kubectl 任务分析:**
- 调用了 14 次工具(太多了!)
- Token 线性增长: 3.9k → 7.9k → 12.1k → ... → 76.6k
- 原因: AI 逐个查询每个命名空间,而不是一次性获取

**应该的方式:**
```bash
# ❌ AI 实际做法 (14步)
1. kubectl get pods -A
2. kubectl get nodes
3. kubectl get namespaces
4. kubectl get pods -n default
5. kubectl get pods -n kube-system
6. kubectl get pods -n monitor
... (太多步骤)

# ✅ 理想做法 (1-2步)
1. kubectl get pods -A  # 一次获取所有
2. kubectl get nodes,namespaces  # 如需要额外信息
```

---

## 🔧 优化建议

### 优先级1: 让 LLM 学习到修正 (High Priority)

**方案A: 修改返回结果包含警告**

```go
// 当前实现
if canCorrect {
    // 自动修正成功,继续执行
    tool.Execute(retryCtx, correctedArgs)
    return result.Output  // ❌ LLM 看不到警告
}

// 优化方案
if canCorrect {
    tool.Execute(retryCtx, correctedArgs)

    // 在返回结果中包含警告
    warningMsg := fmt.Sprintf(
        "⚠️ [自动修正] 原始调用 'bash %s' 不合规,已自动修正为使用专用 %s 工具。\n"+
        "💡 提示: 请直接使用 %s 工具,不要通过 bash 调用。\n\n"+
        "执行结果:\n%s",
        args["command"], correctedTool, correctedTool, result.Output,
    )
    return warningMsg  // ✅ LLM 能看到警告并学习
}
```

**效果:**
LLM 会在下一步推理时看到警告,并调整策略,避免重复错误。

---

**方案B: 拒绝执行,要求 LLM 重新调用**

```go
if !isValid {
    // 不自动修正,直接拒绝
    return fmt.Sprintf(
        "❌ 工具调用被拒绝: %s\n"+
        "💡 建议: %s\n"+
        "请使用正确的工具重新调用。",
        errMsg, suggestion,
    )
}
```

**优点:**
- 强制 LLM 学习正确的调用方式
- 不会有"隐形修正"的问题

**缺点:**
- 增加一轮调用 (多消耗 ~5k tokens)
- 用户体验略差 (需要等待重试)

---

**推荐: 方案A (修改返回结果包含警告)**
既保证了执行成功,又让 LLM 学习到了错误。

---

### 优先级2: 扩展 bash blacklist

**当前问题:**
只检测行首和管道后的 kubectl/docker,不检测嵌入在脚本中的。

**优化方案:**
使用更宽松的匹配规则:

```go
func (v *ToolCallValidator) ValidateToolCall(toolName string, args map[string]interface{}) (bool, string, string) {
    if toolName == "bash" {
        command, ok := args["command"].(string)
        if !ok {
            return true, "", ""
        }

        // 检测命令中是否包含禁止的工具关键字
        forbiddenTools := []string{"kubectl", "docker", "redis-cli", "mysql", "psql"}
        for _, tool := range forbiddenTools {
            if strings.Contains(command, tool) {
                return false,
                    fmt.Sprintf("违反 opsxcli 铁律: bash 命令中包含 %s 调用", tool),
                    fmt.Sprintf("请使用专用的 %s 工具,而不是 bash 命令", tool)
            }
        }
    }
    return true, "", ""
}
```

**效果:**
所有包含 kubectl/docker 的 bash 命令都会被拦截,无论是否在行首。

**风险:**
可能拦截合法的命令,如:
```bash
# 可能被误拦截
which kubectl  # 检查 kubectl 是否安装
echo "kubectl version: ..."  # 输出文本
```

**缓解方案:**
添加白名单例外:
```go
// 允许的场景
if strings.HasPrefix(command, "which kubectl") ||
   strings.HasPrefix(command, "command -v kubectl") ||
   strings.Contains(command, "echo") && !regexp.MustCompile(`\$\(`).MatchString(command) {
    return true, "", "" // 允许通过
}
```

---

### 优先级3: 优化 System Prompt 强化禁令

**当前 System Prompt:**
```
⚠️ 重要: kubectl/docker 等有专用工具,不要用 bash 调用!

✅ 示例:
查K8s: kubectl get pods -A (用kubectl工具,不是bash)
监控: uptime && free -h && df -h (bash合并命令)
```

**优化后:**
```
⚠️ 铁律 (违反将被拦截):
- 禁止在 bash 中调用: kubectl, docker, redis-cli, mysql, psql
- 错误示例: bash "kubectl get pods" ❌
- 正确示例: kubectl get pods -A ✅

内置工具 (直接调用,不要用bash):
- kubectl: K8s操作 (kubectl get pods -A)
- docker: 容器管理 (docker ps)
- redis/mysql/psql: 数据库客户端
- bash: 仅用于没有专用工具的命令 (uptime, free, df 等)

⚠️ 特别注意:
- for 循环中不要调用 kubectl ❌
- 命令替换 $(kubectl ...) 中不要用 ❌
- 直接使用 kubectl 工具,传递完整的子命令 ✅
```

**效果:**
明确禁止的场景,减少 AI 犯错的概率。

---

### 优先级4: 引导 AI 合并查询

**问题:**
kubectl 任务用了 14 步,太多了。

**优化方案:**
在 System Prompt 中添加 kubectl 特定示例:

```
kubectl 使用示例:
- 查所有Pod: kubectl get pods -A (一次获取,不要逐个命名空间查询) ✅
- 查节点和命名空间: kubectl get nodes,namespaces ✅
- 避免: 先查命名空间列表,再逐个查询Pod ❌
```

**效果:**
引导 AI 使用更高效的查询方式。

---

## 📋 实施计划

### 立即可做 (30分钟):

1. **修改 `executor.go` 返回警告信息** ✅
   ```go
   if canCorrect {
       // 执行修正后的工具
       result, err := tool.Execute(retryCtx, correctedArgs)

       // 在结果中添加警告
       warningMsg := fmt.Sprintf(
           "⚠️ [自动修正] 不合规调用已修正\n"+
           "原始: bash %v\n"+
           "修正为: %s 工具\n"+
           "💡 下次请直接使用 %s 工具\n\n"+
           "执行结果:\n%s",
           args, correctedTool, correctedTool, result.Output,
       )
       return warningMsg
   }
   ```

2. **扩展 bash blacklist 为包含检测** ✅
   ```go
   forbiddenTools := []string{"kubectl", "docker", "redis-cli"}
   for _, tool := range forbiddenTools {
       if strings.Contains(command, tool) {
           // 检查是否是白名单场景
           if !isWhitelisted(command, tool) {
               return false, errMsg, suggestion
           }
       }
   }
   ```

3. **优化 System Prompt 添加禁令示例** ✅

---

### 后续优化 (1-2小时):

4. **添加统计监控**
   ```go
   // 记录每次修正的详情
   type CorrectionLog struct {
       Timestamp   time.Time
       OriginalTool string
       OriginalArgs map[string]interface{}
       CorrectedTool string
       CorrectedArgs map[string]interface{}
   }

   // 在会话结束时输出统计
   fmt.Printf("\n📊 工具调用统计:\n")
   fmt.Printf("  总调用: %d\n", stats.TotalCalls)
   fmt.Printf("  自动修正: %d (%.1f%%)\n", stats.AutoCorrected, percentage)
   ```

5. **添加学习机制**
   - 记录常见的违规模式
   - 动态调整 System Prompt
   - 提供最佳实践建议

---

## ✅ 验证效果

### 预期改善:

| 指标 | 当前 | 优化后 | 改善 |
|-----|------|--------|------|
| **kubectl Token** | 76.6k | ~10k | -87% |
| **kubectl 步骤** | 14步 | 1-2步 | -85% |
| **自动修正次数** | 每任务1-5次 | 0次 | -100% |
| **LLM 学习率** | 0% (不学习) | 100% (看到警告) | +100% |

---

## 🎯 总结

### ✅ 已验证:
- Validator 机制工作正常
- 能够捕获和修正违规调用
- 显示清晰的警告信息

### ⚠️ 发现的问题:
1. **LLM 未学习到修正** (Critical)
2. **bash blacklist 不够全面**
3. **Token 使用仍然过高**
4. **kubectl 查询策略低效**

### 🚀 优化方向:
1. 修改返回结果包含警告 (让 LLM 学习)
2. 扩展 blacklist 为包含检测
3. 强化 System Prompt 禁令
4. 添加 kubectl 高效查询示例

---

**测试结论:**
Validator **成功验证**,但需要进一步优化让 LLM 真正学习并改变行为。

**下一步:** 实施优先级1-3的优化措施。

---

**最后更新:** 2025-12-21
**版本:** OpsX CLI v1.0.3+ (Validator Tested)
