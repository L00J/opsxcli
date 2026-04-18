package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

// BackgroundTask 后台任务
type BackgroundTask struct {
	ID          string
	Description string
	Query       string
	Status      TaskStatus
	StartTime   time.Time
	Agent       *Agent
	Context     context.Context
	Cancel      context.CancelFunc
	Output      []string
	Error       error
	mutex       sync.RWMutex
}

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskManager 后台任务管理器
type TaskManager struct {
	tasks       []*BackgroundTask
	currentTask int
	mutex       sync.RWMutex
}

// NewTaskManager 创建任务管理器
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:       make([]*BackgroundTask, 0),
		currentTask: -1,
	}
}

// AddTask 添加后台任务
func (tm *TaskManager) AddTask(task *BackgroundTask) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.tasks = append(tm.tasks, task)
	if tm.currentTask == -1 {
		tm.currentTask = 0
	}
}

// RemoveTask 移除完成的任务
func (tm *TaskManager) RemoveTask(taskID string) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	for i, task := range tm.tasks {
		if task.ID == taskID {
			tm.tasks = append(tm.tasks[:i], tm.tasks[i+1:]...)
			if tm.currentTask >= len(tm.tasks) && len(tm.tasks) > 0 {
				tm.currentTask = len(tm.tasks) - 1
			} else if len(tm.tasks) == 0 {
				tm.currentTask = -1
			}
			break
		}
	}
}

// SwitchTask 切换到下一个任务 (循环)
func (tm *TaskManager) SwitchTask() *BackgroundTask {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if len(tm.tasks) == 0 {
		return nil
	}

	tm.currentTask = (tm.currentTask + 1) % len(tm.tasks)
	return tm.tasks[tm.currentTask]
}

// GetCurrentTask 获取当前任务
func (tm *TaskManager) GetCurrentTask() *BackgroundTask {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if tm.currentTask < 0 || tm.currentTask >= len(tm.tasks) {
		return nil
	}

	return tm.tasks[tm.currentTask]
}

// GetAllTasks 获取所有任务
func (tm *TaskManager) GetAllTasks() []*BackgroundTask {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	tasks := make([]*BackgroundTask, len(tm.tasks))
	copy(tasks, tm.tasks)
	return tasks
}

// TaskCount 获取任务数量
func (tm *TaskManager) TaskCount() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	return len(tm.tasks)
}

// RunningTaskCount 获取运行中的任务数量
func (tm *TaskManager) RunningTaskCount() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	count := 0
	for _, task := range tm.tasks {
		if task.Status == TaskStatusRunning {
			count++
		}
	}
	return count
}

// ShowBottomBar 显示底部状态栏 (Claude CLI 风格)
func (tm *TaskManager) ShowBottomBar() {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	taskCount := len(tm.tasks)
	if taskCount == 0 {
		return
	}

	gray := color.New(color.FgHiBlack).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	var statusLine string
	if taskCount == 1 {
		statusLine = fmt.Sprintf("\n%s 1 background task", gray("⏵⏵"))
	} else {
		statusLine = fmt.Sprintf("\n%s %s (shift+tab to cycle) · %d background tasks",
			gray("⏵⏵"),
			green(fmt.Sprintf("Task %d/%d", tm.currentTask+1, taskCount)),
			taskCount)
	}

	fmt.Print(statusLine)
}

// ShowTaskList 显示任务列表
func (tm *TaskManager) ShowTaskList() {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if len(tm.tasks) == 0 {
		fmt.Println("No background tasks")
		return
	}

	separator := color.New(color.FgHiBlack).Sprint("────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────")
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()

	fmt.Println()
	fmt.Println(separator)
	fmt.Printf(" %s\n\n", cyan("Background Tasks"))

	for i, task := range tm.tasks {
		// 任务编号和状态
		prefix := "  "
		if i == tm.currentTask {
			prefix = yellow("❯ ")
		}

		var statusText string
		switch task.Status {
		case TaskStatusRunning:
			duration := time.Since(task.StartTime).Round(time.Second)
			statusText = green(fmt.Sprintf("Running (%s)", duration))
		case TaskStatusCompleted:
			statusText = green("✓ Completed")
		case TaskStatusFailed:
			statusText = red("✗ Failed")
		case TaskStatusCancelled:
			statusText = gray("⊘ Cancelled")
		}

		fmt.Printf("%s%d. %s - %s\n", prefix, i+1, task.Description, statusText)
		fmt.Printf("   %s\n", gray(fmt.Sprintf("Query: %s", truncateString(task.Query, 80))))
	}

	fmt.Println()
	fmt.Printf(" %s\n", gray("Press shift+tab to cycle through tasks"))
	fmt.Println(separator)
	fmt.Println()
}

// AppendOutput 追加任务输出
func (bt *BackgroundTask) AppendOutput(line string) {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	bt.Output = append(bt.Output, line)
}

// GetOutput 获取任务输出
func (bt *BackgroundTask) GetOutput() []string {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	output := make([]string, len(bt.Output))
	copy(output, bt.Output)
	return output
}

// SetStatus 设置任务状态
func (bt *BackgroundTask) SetStatus(status TaskStatus) {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	bt.Status = status
}

// GetStatus 获取任务状态
func (bt *BackgroundTask) GetStatus() TaskStatus {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	return bt.Status
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
