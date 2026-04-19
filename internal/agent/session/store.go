package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"opsxcli/internal/llm"
	"github.com/google/uuid"
)

// JSONLStore JSONL 存储引擎
type JSONLStore struct {
	baseDir string // 会话文件存储目录
}

// NewJSONLStore 创建 JSONL 存储引擎
func NewJSONLStore(baseDir string) (*JSONLStore, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("创建会话存储目录失败: %w", err)
	}

	return &JSONLStore{baseDir: baseDir}, nil
}

// sessionPath 获取会话文件路径
func (s *JSONLStore) sessionPath(sessionID string) string {
	return filepath.Join(s.baseDir, sessionID+".jsonl")
}

// Create 创建新会话文件，写入 meta 记录
func (s *JSONLStore) Create(title, provider, model string) (*Session, error) {
	// 生成唯一会话 ID
	sessionID := "sess_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]

	now := time.Now()
	session := &Session{
		ID:           sessionID,
		Title:        title,
		Provider:     provider,
		Model:        model,
		CreatedAt:    now,
		UpdatedAt:    now,
		MessageCount: 0,
	}

	// 创建并写入 meta 记录
	meta := JSONLRecord{
		Type:      "meta",
		Timestamp: now,
		ID:        sessionID,
		Title:     title,
		Provider:  provider,
		Model:     model,
		CreatedAt: now,
		UpdatedAt: now,
	}

	filePath := s.sessionPath(sessionID)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("创建会话文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(meta); err != nil {
		return nil, fmt.Errorf("写入 meta 记录失败: %w", err)
	}

	return session, nil
}

// WriteMessage 写入一条消息记录
func (s *JSONLStore) WriteMessage(sessionID string, msg llm.Message) error {
	record := JSONLRecord{
		Type:       "message",
		Timestamp:  time.Now(),
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCallID: msg.ToolCallID,
		Name:       msg.Name,
	}

	// 转换 ToolCalls
	if len(msg.ToolCalls) > 0 {
		record.ToolCalls = make([]ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			record.ToolCalls[i] = ToolCall{
				ID: tc.ID,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}

	filePath := s.sessionPath(sessionID)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("打开会话文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("写入消息记录失败: %w", err)
	}

	return nil
}

// WriteToolResult 写入一条工具结果记录
func (s *JSONLStore) WriteToolResult(sessionID string, toolCallID, name string, success bool, output, errStr string) error {
	record := JSONLRecord{
		Type:        "tool_result",
		Timestamp:   time.Now(),
		ToolCallID_: toolCallID,
		Name_:       name,
		Success:     success,
		Output:      output,
		Error:       errStr,
	}

	filePath := s.sessionPath(sessionID)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("打开会话文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(record); err != nil {
		return fmt.Errorf("写入工具结果记录失败: %w", err)
	}

	return nil
}

// LoadMessages 加载会话的所有消息（转换为 llm.Message 格式）
func (s *JSONLStore) LoadMessages(sessionID string) ([]llm.Message, error) {
	records, err := s.loadAllRecords(sessionID)
	if err != nil {
		return nil, err
	}

	var messages []llm.Message
	for _, record := range records {
		if record.Type != "message" {
			continue
		}

		msg := llm.Message{
			Role:       record.Role,
			Content:    record.Content,
			ToolCallID: record.ToolCallID,
			Name:       record.Name,
		}

		// 转换 ToolCalls
		if len(record.ToolCalls) > 0 {
			msg.ToolCalls = make([]llm.ToolCall, len(record.ToolCalls))
			for i, tc := range record.ToolCalls {
				msg.ToolCalls[i] = llm.ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: llm.FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// LoadSession 加载会话元信息
func (s *JSONLStore) LoadSession(sessionID string) (*Session, error) {
	records, err := s.loadAllRecords(sessionID)
	if err != nil {
		return nil, err
	}

	// 查找 meta 记录
	for _, record := range records {
		if record.Type == "meta" {
			msgCount := 0
			for _, r := range records {
				if r.Type == "message" {
					msgCount++
				}
			}

			return &Session{
				ID:           record.ID,
				Title:        record.Title,
				Provider:     record.Provider,
				Model:        record.Model,
				CreatedAt:    record.CreatedAt,
				UpdatedAt:    record.UpdatedAt,
				MessageCount: msgCount,
			}, nil
		}
	}

	return nil, fmt.Errorf("会话 %s 不存在或已损坏", sessionID)
}

// List 列出所有会话（按时间倒序）
func (s *JSONLStore) List(limit int) ([]*Session, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("读取会话目录失败: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
		session, err := s.LoadSession(sessionID)
		if err != nil {
			// 跳过损坏的会话文件
			continue
		}

		sessions = append(sessions, session)
	}

	// 按更新时间倒序排列
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	// 限制数量
	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

// Delete 删除会话文件
func (s *JSONLStore) Delete(sessionID string) error {
	filePath := s.sessionPath(sessionID)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("会话 %s 不存在", sessionID)
		}
		return fmt.Errorf("删除会话文件失败: %w", err)
	}
	return nil
}

// ExportJSON 导出会话为 JSON 格式
func (s *JSONLStore) ExportJSON(sessionID string) (string, error) {
	session, err := s.LoadSession(sessionID)
	if err != nil {
		return "", err
	}

	records, err := s.loadAllRecords(sessionID)
	if err != nil {
		return "", err
	}

	type exportJSONSession struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Provider  string    `json:"provider"`
		Model     string    `json:"model"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	type exportJSONMessage struct {
		Role       string     `json:"role"`
		Content    string     `json:"content"`
		Name       string     `json:"name,omitempty"`
		ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
		ToolCallID string     `json:"tool_call_id,omitempty"`
	}

	type exportJSONToolResult struct {
		ToolCallID string `json:"tool_call_id"`
		Name       string `json:"name"`
		Success    bool   `json:"success"`
		Output     string `json:"output,omitempty"`
		Error      string `json:"error,omitempty"`
	}

	type exportJSONStatistics struct {
		TotalMessages     int `json:"total_messages"`
		UserMessages      int `json:"user_messages"`
		AssistantMessages int `json:"assistant_messages"`
		ToolCalls         int `json:"tool_calls"`
	}

	type exportJSONData struct {
		Session     exportJSONSession      `json:"session"`
		Messages    []exportJSONMessage    `json:"messages"`
		ToolResults []exportJSONToolResult `json:"tool_results"`
		Statistics  exportJSONStatistics   `json:"statistics"`
	}

	data := exportJSONData{
		Session: exportJSONSession{
			ID:        session.ID,
			Title:     session.Title,
			Provider:  session.Provider,
			Model:     session.Model,
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
		},
	}

	stats := exportJSONStatistics{}

	for _, record := range records {
		switch record.Type {
		case "message":
			msg := exportJSONMessage{
				Role:       record.Role,
				Content:    record.Content,
				Name:       record.Name,
				ToolCallID: record.ToolCallID,
			}
			if len(record.ToolCalls) > 0 {
				msg.ToolCalls = record.ToolCalls
				stats.ToolCalls += len(record.ToolCalls)
			}
			data.Messages = append(data.Messages, msg)
			stats.TotalMessages++
			switch record.Role {
			case "user":
				stats.UserMessages++
			case "assistant":
				stats.AssistantMessages++
			}
		case "tool_result":
			data.ToolResults = append(data.ToolResults, exportJSONToolResult{
				ToolCallID: record.ToolCallID_,
				Name:       record.Name_,
				Success:    record.Success,
				Output:     record.Output,
				Error:      record.Error,
			})
		}
	}

	data.Statistics = stats

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	return string(bytes), nil
}

// ExportToFile 导出会话到文件
// 返回实际写入的文件路径
func (s *JSONLStore) ExportToFile(sessionID, format, filePath string) (string, error) {
	var content string
	var ext string
	var err error

	switch format {
	case "markdown", "md":
		content, err = s.ExportMarkdown(sessionID)
		ext = "md"
	case "json":
		content, err = s.ExportJSON(sessionID)
		ext = "json"
	default:
		return "", fmt.Errorf("不支持的导出格式: %s", format)
	}

	if err != nil {
		return "", err
	}

	// 如果文件路径为空，自动生成
	if filePath == "" {
		session, err := s.LoadSession(sessionID)
		if err != nil {
			return "", err
		}
		safeTitle := sanitizeFilename(session.Title)
		timestamp := time.Now().Format("20060102_150405")
		filePath = fmt.Sprintf("%s_%s.%s", safeTitle, timestamp, ext)
	}

	if err := os.WriteFile(filePath, []byte(content), 0640); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return filePath, nil
}

// sanitizeFilename 清理文件名中的特殊字符
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}

// ExportMarkdown 导出会话为 Markdown 格式
func (s *JSONLStore) ExportMarkdown(sessionID string) (string, error) {
	records, err := s.loadAllRecords(sessionID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	var session *Session

	for _, record := range records {
		if record.Type == "meta" {
			session = &Session{
				ID:        record.ID,
				Title:     record.Title,
				Provider:  record.Provider,
				Model:     record.Model,
				CreatedAt: record.CreatedAt,
				UpdatedAt: record.UpdatedAt,
			}
			break
		}
	}

	if session != nil {
		sb.WriteString(fmt.Sprintf("# %s\n\n", session.Title))
		sb.WriteString(fmt.Sprintf("- **会话ID**: %s\n", session.ID))
		sb.WriteString(fmt.Sprintf("- **提供商**: %s\n", session.Provider))
		sb.WriteString(fmt.Sprintf("- **模型**: %s\n", session.Model))
		sb.WriteString(fmt.Sprintf("- **创建时间**: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05")))
		sb.WriteString("\n---\n\n")
	}

	for _, record := range records {
		switch record.Type {
		case "message":
			switch record.Role {
			case "user":
				sb.WriteString(fmt.Sprintf("## 用户\n\n%s\n\n", record.Content))
			case "assistant":
				sb.WriteString(fmt.Sprintf("## Agent\n\n%s\n\n", record.Content))
				if len(record.ToolCalls) > 0 {
					sb.WriteString("**工具调用**:\n\n")
					for _, tc := range record.ToolCalls {
						sb.WriteString(fmt.Sprintf("- `%s`(%s)\n", tc.Function.Name, tc.ID))
						sb.WriteString(fmt.Sprintf("  ```json\n  %s\n  ```\n\n", tc.Function.Arguments))
					}
				}
			case "system":
				sb.WriteString(fmt.Sprintf("<!-- 系统指令: %s -->\n\n", record.Content))
			case "tool":
				sb.WriteString(fmt.Sprintf("## 工具结果 (%s)\n\n%s\n\n", record.Name, record.Content))
			}
		case "tool_result":
			status := "成功"
			if !record.Success {
				status = "失败"
			}
			sb.WriteString(fmt.Sprintf("### 工具执行结果 [%s]\n\n", status))
			sb.WriteString(fmt.Sprintf("- **工具**: %s\n", record.Name_))
			sb.WriteString(fmt.Sprintf("- **调用ID**: %s\n", record.ToolCallID_))
			if record.Output != "" {
				sb.WriteString(fmt.Sprintf("\n```\n%s\n```\n\n", record.Output))
			}
			if record.Error != "" {
				sb.WriteString(fmt.Sprintf("\n**错误**: %s\n\n", record.Error))
			}
		}
	}

	return sb.String(), nil
}

// loadAllRecords 加载会话的所有记录
func (s *JSONLStore) loadAllRecords(sessionID string) ([]JSONLRecord, error) {
	filePath := s.sessionPath(sessionID)
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("会话 %s 不存在", sessionID)
		}
		return nil, fmt.Errorf("打开会话文件失败: %w", err)
	}
	defer file.Close()

	var records []JSONLRecord
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var record JSONLRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			// 跳过损坏的行，继续处理
			continue
		}
		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取会话文件失败: %w", err)
	}

	return records, nil
}

// UpdateMetaTimestamp 更新会话的 meta 记录时间戳
func (s *JSONLStore) UpdateMetaTimestamp(sessionID string) error {
	records, err := s.loadAllRecords(sessionID)
	if err != nil {
		return err
	}

	// 找到并更新 meta 记录
	now := time.Now()
	for i := range records {
		if records[i].Type == "meta" {
			records[i].UpdatedAt = now
			break
		}
	}

	// 重写整个文件
	filePath := s.sessionPath(sessionID)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("打开会话文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("写入记录失败: %w", err)
		}
	}

	return nil
}

// UpdateTitle 更新会话标题
func (s *JSONLStore) UpdateTitle(sessionID, title string) error {
	records, err := s.loadAllRecords(sessionID)
	if err != nil {
		return err
	}

	for i := range records {
		if records[i].Type == "meta" {
			records[i].Title = title
			records[i].UpdatedAt = time.Now()
			break
		}
	}

	filePath := s.sessionPath(sessionID)
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("打开会话文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("写入记录失败: %w", err)
		}
	}

	return nil
}
