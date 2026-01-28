package tui

import (
	"context"
	"fmt"
	"sync"
)

// SessionState 会话状态
type SessionState string

const (
	SessionStateIdle    SessionState = "idle"    // 空闲
	SessionStateRunning SessionState = "running" // 运行中
	SessionStateWaiting SessionState = "waiting" // 等待输入
	SessionStateDone    SessionState = "done"    // 完成
)

// SessionMessage 会话消息
type SessionMessage struct {
	Role    string // user, assistant, system
	Content string
}

// InteractiveSession 交互式会话
type InteractiveSession struct {
	mu           sync.RWMutex
	state        SessionState
	messages     []SessionMessage
	inputChan    chan string
	outputChan   chan string
	errorChan    chan error
	cancelFunc   context.CancelFunc
	allowInput   bool // 是否允许在运行时输入
}

// NewInteractiveSession 创建交互式会话
func NewInteractiveSession(allowInput bool) *InteractiveSession {
	return &InteractiveSession{
		state:      SessionStateIdle,
		messages:   make([]SessionMessage, 0),
		inputChan:  make(chan string, 10),
		outputChan: make(chan string, 10),
		errorChan:  make(chan error, 10),
		allowInput: allowInput,
	}
}

// Start 启动会话
func (s *InteractiveSession) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionStateIdle {
		return fmt.Errorf("会话已启动")
	}

	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.state = SessionStateRunning

	// 如果允许运行时输入,启动输入监听
	if s.allowInput {
		go s.listenForInput(ctx)
	}

	return nil
}

// listenForInput 监听用户输入
func (s *InteractiveSession) listenForInput(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 显示输入提示
			input, err := RunInput(">")
			if err != nil {
				// 用户取消或错误,继续等待
				continue
			}

			// 发送输入到通道
			select {
			case s.inputChan <- input:
				s.AddMessage("user", input)
			case <-ctx.Done():
				return
			}
		}
	}
}

// AddMessage 添加消息
func (s *InteractiveSession) AddMessage(role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append(s.messages, SessionMessage{
		Role:    role,
		Content: content,
	})
}

// GetMessages 获取所有消息
func (s *InteractiveSession) GetMessages() []SessionMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本
	messages := make([]SessionMessage, len(s.messages))
	copy(messages, s.messages)
	return messages
}

// WaitForInput 等待用户输入
func (s *InteractiveSession) WaitForInput(prompt string) (string, error) {
	s.mu.Lock()
	oldState := s.state
	s.state = SessionStateWaiting
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.state = oldState
		s.mu.Unlock()
	}()

	// 如果已经有输入在通道中,直接返回
	select {
	case input := <-s.inputChan:
		return input, nil
	default:
		// 没有现成的输入,显示提示并等待
		input, err := RunInput(prompt)
		if err != nil {
			return "", err
		}
		s.AddMessage("user", input)
		return input, nil
	}
}

// SendOutput 发送输出
func (s *InteractiveSession) SendOutput(content string) {
	s.AddMessage("assistant", content)
	select {
	case s.outputChan <- content:
	default:
		// 输出通道已满,忽略
	}
}

// SendError 发送错误
func (s *InteractiveSession) SendError(err error) {
	select {
	case s.errorChan <- err:
	default:
		// 错误通道已满,忽略
	}
}

// Stop 停止会话
func (s *InteractiveSession) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	s.state = SessionStateDone
}

// GetState 获取当前状态
func (s *InteractiveSession) GetState() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// IsRunning 是否正在运行
func (s *InteractiveSession) IsRunning() bool {
	return s.GetState() == SessionStateRunning
}

// IsWaiting 是否等待输入
func (s *InteractiveSession) IsWaiting() bool {
	return s.GetState() == SessionStateWaiting
}

// IsDone 是否已完成
func (s *InteractiveSession) IsDone() bool {
	return s.GetState() == SessionStateDone
}
