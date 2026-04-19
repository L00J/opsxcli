package session

import (
	"os"
	"path/filepath"
	"testing"

	"opsxcli/internal/llm"
)

// TestJSONLStore_Create 测试创建会话
func TestJSONLStore_Create(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("测试会话", "deepseek", "deepseek-chat")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if session.ID == "" {
		t.Error("session.ID should not be empty")
	}
	if session.Title != "测试会话" {
		t.Errorf("session.Title = %q, want %q", session.Title, "测试会话")
	}
	if session.Provider != "deepseek" {
		t.Errorf("session.Provider = %q, want %q", session.Provider, "deepseek")
	}
	if session.Model != "deepseek-chat" {
		t.Errorf("session.Model = %q, want %q", session.Model, "deepseek-chat")
	}
	if session.MessageCount != 0 {
		t.Errorf("session.MessageCount = %d, want 0", session.MessageCount)
	}

	// 验证文件已创建
	filePath := filepath.Join(tmpDir, session.ID+".jsonl")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("session file not created: %s", filePath)
	}
}

// TestJSONLStore_WriteMessageAndLoad 测试写入和加载消息
func TestJSONLStore_WriteMessageAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("消息测试", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// 写入消息
	msg := llm.Message{Role: "user", Content: "你好"}
	if err := store.WriteMessage(session.ID, msg); err != nil {
		t.Fatalf("WriteMessage() error: %v", err)
	}

	msg2 := llm.Message{Role: "assistant", Content: "你好！有什么可以帮你的？"}
	if err := store.WriteMessage(session.ID, msg2); err != nil {
		t.Fatalf("WriteMessage() error: %v", err)
	}

	// 加载消息
	messages, err := store.LoadMessages(session.ID)
	if err != nil {
		t.Fatalf("LoadMessages() error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("LoadMessages() returned %d messages, want 2", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "你好" {
		t.Errorf("messages[0] = %+v, want user/你好", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "你好！有什么可以帮你的？" {
		t.Errorf("messages[1] = %+v, want assistant/你好！有什么可以帮你的？", messages[1])
	}

	// 加载会话元信息，验证消息计数
	loadedSession, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error: %v", err)
	}
	if loadedSession.MessageCount != 2 {
		t.Errorf("MessageCount = %d, want 2", loadedSession.MessageCount)
	}
}

// TestJSONLStore_WriteToolResult 测试写入工具结果
func TestJSONLStore_WriteToolResult(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("工具测试", "claude", "claude-3")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := store.WriteToolResult(session.ID, "call_123", "local_bash", true, "output text", ""); err != nil {
		t.Fatalf("WriteToolResult() error: %v", err)
	}

	// 加载所有记录验证
	records, err := store.loadAllRecords(session.ID)
	if err != nil {
		t.Fatalf("loadAllRecords() error: %v", err)
	}

	var foundToolResult bool
	for _, r := range records {
		if r.Type == "tool_result" && r.ToolCallID_ == "call_123" {
			foundToolResult = true
			if !r.Success {
				t.Error("tool result should be successful")
			}
			if r.Output != "output text" {
				t.Errorf("tool result output = %q, want %q", r.Output, "output text")
			}
		}
	}
	if !foundToolResult {
		t.Error("tool_result record not found")
	}
}

// TestJSONLStore_LoadSession_notFound 测试加载不存在的会话
func TestJSONLStore_LoadSession_notFound(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	_, err = store.LoadSession("nonexistent")
	if err == nil {
		t.Error("LoadSession(nonexistent) expected error, got nil")
	}
}

// TestJSONLStore_List 测试会话列表
func TestJSONLStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	// 创建多个会话
	for i := 0; i < 3; i++ {
		_, err := store.Create("会话", "provider", "model")
		if err != nil {
			t.Fatalf("Create() error: %v", err)
		}
	}

	sessions, err := store.List(10)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("List() returned %d sessions, want 3", len(sessions))
	}

	// 测试 limit
	sessionsLimited, err := store.List(2)
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(sessionsLimited) != 2 {
		t.Errorf("List(2) returned %d sessions, want 2", len(sessionsLimited))
	}
}

// TestJSONLStore_Delete 测试删除会话
func TestJSONLStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("待删除", "test", "test")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// 删除
	if err := store.Delete(session.ID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}

	// 验证文件已删除
	filePath := filepath.Join(tmpDir, session.ID+".jsonl")
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("session file should be deleted: %s", filePath)
	}

	// 删除不存在的会话
	if err := store.Delete("nonexistent"); err == nil {
		t.Error("Delete(nonexistent) expected error, got nil")
	}
}

// TestJSONLStore_ExportMarkdown 测试导出 Markdown
func TestJSONLStore_ExportMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("导出测试", "deepseek", "deepseek-chat")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// 写入消息
	store.WriteMessage(session.ID, llm.Message{Role: "user", Content: "查看磁盘"})
	store.WriteMessage(session.ID, llm.Message{Role: "assistant", Content: "df -h 结果显示..."})

	markdown, err := store.ExportMarkdown(session.ID)
	if err != nil {
		t.Fatalf("ExportMarkdown() error: %v", err)
	}
	if markdown == "" {
		t.Error("ExportMarkdown() returned empty string")
	}
	if !contains(markdown, "导出测试") {
		t.Error("ExportMarkdown() should contain session title")
	}
	if !contains(markdown, "查看磁盘") {
		t.Error("ExportMarkdown() should contain user message")
	}
	if !contains(markdown, "df -h") {
		t.Error("ExportMarkdown() should contain assistant message")
	}
}

// TestJSONLStore_ExportJSON 测试导出 JSON
func TestJSONLStore_ExportJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("JSON导出测试", "deepseek", "deepseek-chat")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// 写入消息
	store.WriteMessage(session.ID, llm.Message{Role: "system", Content: "系统指令"})
	store.WriteMessage(session.ID, llm.Message{Role: "user", Content: "查看磁盘"})
	store.WriteMessage(session.ID, llm.Message{
		Role:      "assistant",
		Content:   "我来查看磁盘使用情况",
		ToolCalls: []llm.ToolCall{{ID: "call_1", Type: "function", Function: llm.FunctionCall{Name: "local_bash", Arguments: "{\"cmd\":\"df -h\"}"}}},
	})
	store.WriteMessage(session.ID, llm.Message{Role: "tool", Content: "磁盘使用结果", ToolCallID: "call_1", Name: "local_bash"})
	store.WriteToolResult(session.ID, "call_1", "local_bash", true, "磁盘正常", "")

	jsonStr, err := store.ExportJSON(session.ID)
	if err != nil {
		t.Fatalf("ExportJSON() error: %v", err)
	}
	if jsonStr == "" {
		t.Fatal("ExportJSON() returned empty string")
	}

	// 验证包含关键内容
	if !contains(jsonStr, "JSON导出测试") {
		t.Error("ExportJSON() should contain session title")
	}
	if !contains(jsonStr, "sess_") {
		t.Error("ExportJSON() should contain session id")
	}
	if !contains(jsonStr, "deepseek") {
		t.Error("ExportJSON() should contain provider")
	}
	if !contains(jsonStr, "查看磁盘") {
		t.Error("ExportJSON() should contain user message")
	}
	if !contains(jsonStr, "local_bash") {
		t.Error("ExportJSON() should contain tool name")
	}
	if !contains(jsonStr, "磁盘正常") {
		t.Error("ExportJSON() should contain tool output")
	}
	if !contains(jsonStr, "total_messages") {
		t.Error("ExportJSON() should contain statistics")
	}
}

// TestJSONLStore_ExportToFile 测试导出到文件
func TestJSONLStore_ExportToFile(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("文件导出测试", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	store.WriteMessage(session.ID, llm.Message{Role: "user", Content: "hello"})

	// 测试 Markdown 导出到指定文件
	mdPath := filepath.Join(tmpDir, "test_export.md")
	path, err := store.ExportToFile(session.ID, "markdown", mdPath)
	if err != nil {
		t.Fatalf("ExportToFile(markdown) error: %v", err)
	}
	if path != mdPath {
		t.Errorf("ExportToFile() path = %q, want %q", path, mdPath)
	}
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if !contains(string(content), "hello") {
		t.Error("Markdown file should contain user message")
	}

	// 测试 JSON 导出到指定文件
	jsonPath := filepath.Join(tmpDir, "test_export.json")
	path, err = store.ExportToFile(session.ID, "json", jsonPath)
	if err != nil {
		t.Fatalf("ExportToFile(json) error: %v", err)
	}
	if path != jsonPath {
		t.Errorf("ExportToFile() path = %q, want %q", path, jsonPath)
	}
	content, err = os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if !contains(string(content), "hello") {
		t.Error("JSON file should contain user message")
	}

	// 测试自动生成文件名
	path, err = store.ExportToFile(session.ID, "json", "")
	if err != nil {
		t.Fatalf("ExportToFile(auto) error: %v", err)
	}
	if path == "" {
		t.Error("ExportToFile(auto) should return generated filename")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("ExportToFile(auto) file not created: %s", path)
	}

	// 测试不支持的格式
	_, err = store.ExportToFile(session.ID, "xml", "")
	if err == nil {
		t.Error("ExportToFile(xml) expected error, got nil")
	}
}

// TestJSONLStore_UpdateTitle 测试更新标题
func TestJSONLStore_UpdateTitle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewJSONLStore(tmpDir)
	if err != nil {
		t.Fatalf("NewJSONLStore() error: %v", err)
	}

	session, err := store.Create("旧标题", "test", "test")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	if err := store.UpdateTitle(session.ID, "新标题"); err != nil {
		t.Fatalf("UpdateTitle() error: %v", err)
	}

	loaded, err := store.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("LoadSession() error: %v", err)
	}
	if loaded.Title != "新标题" {
		t.Errorf("Title = %q, want %q", loaded.Title, "新标题")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
