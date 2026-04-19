package session

import (
	"strings"
	"testing"

	"opsxcli/internal/llm"
)

func TestManagerCreate(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("test title", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if sess == nil {
		t.Fatal("expected session, got nil")
	}
	if !strings.HasPrefix(sess.ID, "sess_") {
		t.Errorf("expected ID to start with 'sess_', got %q", sess.ID)
	}
	if sess.Title != "test title" {
		t.Errorf("expected title %q, got %q", "test title", sess.Title)
	}
	if sess.Provider != "openai" {
		t.Errorf("expected provider %q, got %q", "openai", sess.Provider)
	}
	if sess.Model != "gpt-4" {
		t.Errorf("expected model %q, got %q", "gpt-4", sess.Model)
	}
	if sess.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if sess.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if sess.MessageCount != 0 {
		t.Errorf("expected message count 0, got %d", sess.MessageCount)
	}
}

func TestManagerCreateEmptyTitle(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if sess.Title != "新会话" {
		t.Errorf("expected default title %q, got %q", "新会话", sess.Title)
	}
}

func TestManagerSaveMessage(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("msg test", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	userMsg := llm.Message{Role: "user", Content: "hello"}
	if err := m.SaveMessage(sess.ID, userMsg); err != nil {
		t.Fatalf("save user message: %v", err)
	}

	assistantMsg := llm.Message{Role: "assistant", Content: "hi there"}
	if err := m.SaveMessage(sess.ID, assistantMsg); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}

	loaded, messages, err := m.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.MessageCount != 2 {
		t.Errorf("expected message count 2, got %d", loaded.MessageCount)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "hello" {
		t.Errorf("expected first message user/hello, got %s/%s", messages[0].Role, messages[0].Content)
	}
	if messages[1].Role != "assistant" || messages[1].Content != "hi there" {
		t.Errorf("expected second message assistant/hi there, got %s/%s", messages[1].Role, messages[1].Content)
	}
}

func TestManagerSaveToolResult(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("tool test", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := m.SaveToolResult(sess.ID, "call_123", "my_tool", true, "output data", ""); err != nil {
		t.Fatalf("save tool result: %v", err)
	}

	loaded, messages, err := m.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	// tool_result records are not counted as messages by LoadMessages
	if loaded.MessageCount != 0 {
		t.Errorf("expected message count 0, got %d", loaded.MessageCount)
	}
	if len(messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(messages))
	}

	// Verify by loading raw records through store
	records, err := store.loadAllRecords(sess.ID)
	if err != nil {
		t.Fatalf("load records: %v", err)
	}
	var found bool
	for _, r := range records {
		if r.Type == "tool_result" {
			found = true
			if r.ToolCallID_ != "call_123" {
				t.Errorf("expected tool call id %q, got %q", "call_123", r.ToolCallID_)
			}
			if r.Name_ != "my_tool" {
				t.Errorf("expected name %q, got %q", "my_tool", r.Name_)
			}
			if !r.Success {
				t.Error("expected success to be true")
			}
			if r.Output != "output data" {
				t.Errorf("expected output %q, got %q", "output data", r.Output)
			}
			if r.Error != "" {
				t.Errorf("expected empty error, got %q", r.Error)
			}
		}
	}
	if !found {
		t.Error("expected to find tool_result record")
	}
}

func TestManagerLoad(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("load test", "anthropic", "claude-3")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	msg := llm.Message{Role: "user", Content: "round trip"}
	if err := m.SaveMessage(sess.ID, msg); err != nil {
		t.Fatalf("save message: %v", err)
	}

	loaded, messages, err := m.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.ID != sess.ID {
		t.Errorf("expected id %q, got %q", sess.ID, loaded.ID)
	}
	if loaded.Title != "load test" {
		t.Errorf("expected title %q, got %q", "load test", loaded.Title)
	}
	if loaded.Provider != "anthropic" {
		t.Errorf("expected provider %q, got %q", "anthropic", loaded.Provider)
	}
	if loaded.Model != "claude-3" {
		t.Errorf("expected model %q, got %q", "claude-3", loaded.Model)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Content != "round trip" {
		t.Errorf("expected content %q, got %q", "round trip", messages[0].Content)
	}
}

func TestManagerLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	_, _, err = m.Load("sess_nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestManagerList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	s1, err := m.Create("session one", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create s1: %v", err)
	}
	// Save a message to update s1's UpdatedAt
	if err := m.SaveMessage(s1.ID, llm.Message{Role: "user", Content: "msg1"}); err != nil {
		t.Fatalf("save message to s1: %v", err)
	}

	s2, err := m.Create("session two", "anthropic", "claude-3")
	if err != nil {
		t.Fatalf("create s2: %v", err)
	}
	if err := m.SaveMessage(s2.ID, llm.Message{Role: "user", Content: "msg2"}); err != nil {
		t.Fatalf("save message to s2: %v", err)
	}

	s3, err := m.Create("session three", "google", "gemini")
	if err != nil {
		t.Fatalf("create s3: %v", err)
	}
	if err := m.SaveMessage(s3.ID, llm.Message{Role: "user", Content: "msg3"}); err != nil {
		t.Fatalf("save message to s3: %v", err)
	}

	// List all
	all, err := m.List(0)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(all))
	}

	// Most recently updated should be first (s3, then s2, then s1)
	if all[0].ID != s3.ID {
		t.Errorf("expected first id %q, got %q", s3.ID, all[0].ID)
	}
	if all[1].ID != s2.ID {
		t.Errorf("expected second id %q, got %q", s2.ID, all[1].ID)
	}
	if all[2].ID != s1.ID {
		t.Errorf("expected third id %q, got %q", s1.ID, all[2].ID)
	}

	// List with limit
	limited, err := m.List(2)
	if err != nil {
		t.Fatalf("list limited: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(limited))
	}
	if limited[0].ID != s3.ID {
		t.Errorf("expected first id %q, got %q", s3.ID, limited[0].ID)
	}
	if limited[1].ID != s2.ID {
		t.Errorf("expected second id %q, got %q", s2.ID, limited[1].ID)
	}
}

func TestManagerDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("delete me", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := m.Delete(sess.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	_, _, err = m.Load(sess.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}

	// Delete again should error
	if err := m.Delete(sess.ID); err == nil {
		t.Fatal("expected error deleting nonexistent session")
	}
}

func TestManagerUpdateTitle(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("old title", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := m.UpdateTitle(sess.ID, "new title"); err != nil {
		t.Fatalf("update title: %v", err)
	}

	loaded, _, err := m.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.Title != "new title" {
		t.Errorf("expected title %q, got %q", "new title", loaded.Title)
	}
}

func TestManagerExportMarkdown(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	m := NewManager(store)

	sess, err := m.Create("export test", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := m.SaveMessage(sess.ID, llm.Message{Role: "user", Content: "user question"}); err != nil {
		t.Fatalf("save user message: %v", err)
	}
	if err := m.SaveMessage(sess.ID, llm.Message{Role: "assistant", Content: "assistant answer"}); err != nil {
		t.Fatalf("save assistant message: %v", err)
	}
	if err := m.SaveToolResult(sess.ID, "call_1", "test_tool", true, "tool output", ""); err != nil {
		t.Fatalf("save tool result: %v", err)
	}

	md, err := m.ExportMarkdown(sess.ID)
	if err != nil {
		t.Fatalf("export markdown: %v", err)
	}

	if !strings.Contains(md, "# export test") {
		t.Error("expected markdown to contain title header")
	}
	if !strings.Contains(md, sess.ID) {
		t.Error("expected markdown to contain session ID")
	}
	if !strings.Contains(md, "openai") {
		t.Error("expected markdown to contain provider")
	}
	if !strings.Contains(md, "gpt-4") {
		t.Error("expected markdown to contain model")
	}
	if !strings.Contains(md, "用户") {
		t.Error("expected markdown to contain user section")
	}
	if !strings.Contains(md, "user question") {
		t.Error("expected markdown to contain user question")
	}
	if !strings.Contains(md, "Agent") {
		t.Error("expected markdown to contain agent section")
	}
	if !strings.Contains(md, "assistant answer") {
		t.Error("expected markdown to contain assistant answer")
	}
	if !strings.Contains(md, "工具执行结果") {
		t.Error("expected markdown to contain tool result section")
	}
	if !strings.Contains(md, "test_tool") {
		t.Error("expected markdown to contain tool name")
	}
	if !strings.Contains(md, "tool output") {
		t.Error("expected markdown to contain tool output")
	}
}
