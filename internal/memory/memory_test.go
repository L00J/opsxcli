package memory

import (
	"encoding/json"
	"testing"
	"time"

	"opsxcli/internal/db"
	"opsxcli/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB 创建测试用数据库
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.NewDB(tmpDir)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	return database
}

func TestNewMemoryManager(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)
	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.shortTermCache)
	assert.Equal(t, int64(100*1024*1024), mgr.maxShortTermSize)
	assert.Equal(t, int64(400*1024*1024), mgr.maxLongTermSize)
	assert.Equal(t, int64(500*1024*1024), mgr.totalLimit)
}

// === 短暂记忆测试 ===

func TestGetOrCreateContext_NewSession(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	ctx := mgr.GetOrCreateContext("session-1")
	assert.NotNil(t, ctx)
	assert.Equal(t, "session-1", ctx.SessionID)
	assert.Empty(t, ctx.Messages)
	assert.NotNil(t, ctx.ToolUsage)
	assert.Equal(t, 50, ctx.MaxMessages)
}

func TestGetOrCreateContext_ExistingSession(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	ctx1 := mgr.GetOrCreateContext("session-1")
	ctx2 := mgr.GetOrCreateContext("session-1")
	// 应返回同一实例
	assert.Same(t, ctx1, ctx2)
}

func TestGetOrCreateContext_DifferentSessions(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	ctx1 := mgr.GetOrCreateContext("session-1")
	ctx2 := mgr.GetOrCreateContext("session-2")
	assert.NotSame(t, ctx1, ctx2)
	assert.Equal(t, "session-1", ctx1.SessionID)
	assert.Equal(t, "session-2", ctx2.SessionID)
}

func TestAddMessage(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	msg := llm.Message{Role: "user", Content: "你好"}
	mgr.AddMessage("session-1", msg)

	messages := mgr.GetRecentMessages("session-1", 0)
	assert.Len(t, messages, 1)
	assert.Equal(t, "user", messages[0].Role)
	assert.Equal(t, "你好", messages[0].Content)
}

func TestAddMessage_MultipleMessages(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	mgr.AddMessage("session-1", llm.Message{Role: "user", Content: "消息1"})
	mgr.AddMessage("session-1", llm.Message{Role: "assistant", Content: "回复1"})
	mgr.AddMessage("session-1", llm.Message{Role: "user", Content: "消息2"})

	messages := mgr.GetRecentMessages("session-1", 0)
	assert.Len(t, messages, 3)
}

func TestGetRecentMessages_WithLimit(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	for i := 0; i < 10; i++ {
		mgr.AddMessage("session-1", llm.Message{
			Role:    "user",
			Content: "消息",
		})
	}

	// 获取最近 3 条
	messages := mgr.GetRecentMessages("session-1", 3)
	assert.Len(t, messages, 3)
}

func TestGetRecentMessages_LimitZero_ReturnsAll(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	for i := 0; i < 5; i++ {
		mgr.AddMessage("session-1", llm.Message{Role: "user", Content: "msg"})
	}

	messages := mgr.GetRecentMessages("session-1", 0)
	assert.Len(t, messages, 5)
}

func TestGetRecentMessages_LimitExceedsLength(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	for i := 0; i < 3; i++ {
		mgr.AddMessage("session-1", llm.Message{Role: "user", Content: "msg"})
	}

	// limit 大于消息数，应返回全部
	messages := mgr.GetRecentMessages("session-1", 100)
	assert.Len(t, messages, 3)
}

func TestRecordToolUsage(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	mgr.RecordToolUsage("session-1", "local_bash")
	mgr.RecordToolUsage("session-1", "local_bash")
	mgr.RecordToolUsage("session-1", "ssh_execute")

	ctx := mgr.GetOrCreateContext("session-1")
	assert.Equal(t, 2, ctx.ToolUsage["local_bash"])
	assert.Equal(t, 1, ctx.ToolUsage["ssh_execute"])
}

func TestCompressOldMessages(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	// 添加系统消息
	mgr.AddMessage("session-1", llm.Message{Role: "system", Content: "系统提示"})

	// 添加超过 MaxMessages (50) 的消息来触发压缩
	for i := 0; i < 55; i++ {
		mgr.AddMessage("session-1", llm.Message{
			Role:    "user",
			Content: "消息",
		})
	}

	ctx := mgr.GetOrCreateContext("session-1")
	// 压缩后消息数应减少
	assert.LessOrEqual(t, len(ctx.Messages), 55)
}

func TestGenerateSummary_WithToolCalls(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	messages := []llm.Message{
		{
			Role:    "assistant",
			Content: "执行命令",
			ToolCalls: []llm.ToolCall{
				{Function: llm.FunctionCall{Name: "local_bash"}},
			},
		},
		{
			Role:    "assistant",
			Content: "远程执行",
			ToolCalls: []llm.ToolCall{
				{Function: llm.FunctionCall{Name: "ssh_execute"}},
			},
		},
	}

	summary := mgr.generateSummary(messages)
	assert.Contains(t, summary, "local_bash")
	assert.Contains(t, summary, "ssh_execute")
	assert.Contains(t, summary, "2次工具调用")
}

func TestGenerateSummary_NoToolCalls(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	messages := []llm.Message{
		{Role: "user", Content: "你好"},
		{Role: "assistant", Content: "你好！"},
	}

	summary := mgr.generateSummary(messages)
	assert.Contains(t, summary, "无特殊操作")
}

func TestClearSession(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	mgr.AddMessage("session-1", llm.Message{Role: "user", Content: "msg"})
	mgr.ClearSession("session-1")

	// 清除后应创建新的上下文
	ctx := mgr.GetOrCreateContext("session-1")
	assert.Empty(t, ctx.Messages)
}

// === 永久记忆测试 ===

func TestSaveKnowledgeAndGetKnowledge(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	err := mgr.SaveKnowledge("knowledge", "go-version", "1.24.2")
	assert.NoError(t, err)

	value, err := mgr.GetKnowledge("knowledge", "go-version")
	assert.NoError(t, err)
	assert.Equal(t, "1.24.2", value)
}

func TestGetKnowledge_NotFound(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	_, err := mgr.GetKnowledge("knowledge", "nonexistent")
	assert.Error(t, err)
}

func TestSaveKnowledge_UpdateExisting(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	mgr.SaveKnowledge("knowledge", "go-version", "1.24.0")
	mgr.SaveKnowledge("knowledge", "go-version", "1.24.2")

	value, err := mgr.GetKnowledge("knowledge", "go-version")
	assert.NoError(t, err)
	assert.Equal(t, "1.24.2", value)
}

func TestSavePreferenceAndGetPreference(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	err := mgr.SavePreference("language", "zh-CN")
	assert.NoError(t, err)

	value, err := mgr.GetPreference("language")
	assert.NoError(t, err)
	assert.Equal(t, "zh-CN", value)
}

func TestSaveEnvironmentInfoAndGetEnvironmentInfo(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	err := mgr.SaveEnvironmentInfo("os", "linux")
	assert.NoError(t, err)

	value, err := mgr.GetEnvironmentInfo("os")
	assert.NoError(t, err)
	assert.Equal(t, "linux", value)
}

func TestSaveKnowledge_JSONValue(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	config := map[string]interface{}{
		"host": "localhost",
		"port": 8080,
	}
	jsonValue, _ := json.Marshal(config)

	err := mgr.SaveKnowledge("config", "server", string(jsonValue))
	assert.NoError(t, err)

	value, err := mgr.GetKnowledge("config", "server")
	assert.NoError(t, err)

	var result map[string]interface{}
	json.Unmarshal([]byte(value), &result)
	assert.Equal(t, "localhost", result["host"])
}

func TestCleanExpiredMemories(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	// 保存一个已经过期的记忆（直接插入，expires_at 在过去）
	// 使用 UTC 时间，匹配 SQLite CURRENT_TIMESTAMP 的时区和格式
	past := time.Now().UTC().Add(-1 * time.Hour)
	_, err := database.Exec(`INSERT INTO memories (type, category, key, value, expires_at) VALUES (?, ?, ?, ?, ?)`,
		MemoryTypeLongTerm, "expired", "key1", "value1", past.Format("2006-01-02 15:04:05"))
	require.NoError(t, err)

	// 保存一个不过期的记忆
	err = mgr.SaveKnowledge("valid", "key2", "value2")
	require.NoError(t, err)

	// 清理过期记忆
	err = mgr.CleanExpiredMemories()
	assert.NoError(t, err)

	// 过期记忆应被删除
	_, err = mgr.GetKnowledge("expired", "key1")
	assert.Error(t, err)

	// 有效记忆应保留
	value, err := mgr.GetKnowledge("valid", "key2")
	assert.NoError(t, err)
	assert.Equal(t, "value2", value)
}

func TestGetMemoryStats(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	// 添加短暂记忆
	mgr.AddMessage("session-1", llm.Message{Role: "user", Content: "msg"})

	// 添加永久记忆
	mgr.SaveKnowledge("knowledge", "test", "value")

	stats, err := mgr.GetMemoryStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	assert.Equal(t, 1, stats["short_term_sessions"])
	assert.Equal(t, int64(1), stats["long_term_items"])

	_, ok := stats["total_size_mb"]
	assert.True(t, ok)

	_, ok = stats["usage_percent"]
	assert.True(t, ok)
}

func TestTrimMemoryIfNeeded_BelowLimit(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	mgr.SaveKnowledge("knowledge", "test", "value")
	err := mgr.TrimMemoryIfNeeded()
	assert.NoError(t, err)

	// 在限制以下不应删除任何东西
	value, err := mgr.GetKnowledge("knowledge", "test")
	assert.NoError(t, err)
	assert.Equal(t, "value", value)
}

func TestGetKnowledge_AccessCountIncrement(t *testing.T) {
	database := setupTestDB(t)
	mgr := NewMemoryManager(database)

	mgr.SaveKnowledge("knowledge", "counter", "value")

	// 多次访问
	mgr.GetKnowledge("knowledge", "counter")
	mgr.GetKnowledge("knowledge", "counter")
	mgr.GetKnowledge("knowledge", "counter")

	// 验证访问次数（直接查数据库）
	var count int
	database.QueryRow(`SELECT access_count FROM memories WHERE category = ? AND key = ?`,
		"knowledge", "counter").Scan(&count)
	assert.Equal(t, 3, count)
}
