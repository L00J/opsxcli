package tui

import (
	"opsxcli/internal/agent/session"
	"opsxcli/internal/llm"
)

// streamChunkMsg 流式输出内容片段
type streamChunkMsg struct {
	content string
}

// thinkDoneMsg LLM 思考/流式完成
type thinkDoneMsg struct{}

// toolStartMsg 工具开始执行
type toolStartMsg struct {
	name string
}

// toolDoneMsg 工具执行完成
type toolDoneMsg struct {
	name    string
	success bool
}

// errorMsg 错误消息
type errorMsg struct {
	err error
}

// sessionListMsg 会话列表加载完成
type sessionListMsg struct {
	sessions []*session.Session
	err      error
}

// sessionLoadedMsg 会话加载完成
type sessionLoadedMsg struct {
	session  *session.Session
	messages []llm.Message
	err      error
}

// sessionDeletedMsg 会话删除完成
type sessionDeletedMsg struct {
	sessionID string
	err       error
}

// sessionCreatedMsg 会话创建完成
type sessionCreatedMsg struct {
	session *session.Session
	err     error
}
