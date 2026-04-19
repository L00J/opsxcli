package tui

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
