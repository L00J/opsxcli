package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"opsxcli/internal/agent/evolver"
)

// createTestModelWithDashboard creates a model with dashboard data for testing
func createTestModelWithDashboard() Model {
	m := NewModel(nil, nil)
	m.width = 100
	m.height = 30
	return m
}

func TestRenderDashboard_Initializing(t *testing.T) {
	m := Model{width: 0}
	output := m.renderDashboard()
	assert.Equal(t, "初始化中...", output)
}

func TestRenderDashboard_BasicStructure(t *testing.T) {
	m := createTestModelWithDashboard()
	output := m.renderDashboard()
	assert.Contains(t, output, "控制面板")
	assert.Contains(t, output, "概览")
	assert.Contains(t, output, "技能库")
	assert.Contains(t, output, "切换标签页")
}

func TestRenderDashboard_TabOverview(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabOverview
	output := m.renderDashboard()
	assert.Contains(t, output, "记忆层状态")
	assert.Contains(t, output, "技能库摘要")
}

func TestRenderDashboard_TabSkills(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabSkills
	output := m.renderDashboard()
	assert.Contains(t, output, "暂无技能数据")
}

func TestRenderDashboard_TabSkillsWithData(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabSkills
	m.dashboardData.Skills = []*evolver.SkillEntry{
		{
			ID:          "seed_system_check",
			Name:        "系统健康检查",
			Category:    "system",
			Version:     1,
			Description: "检查系统CPU/内存/磁盘基础指标",
			Steps:       []string{"检查CPU使用率", "检查内存使用率", "检查磁盘空间"},
			Triggers:    []string{"系统检查", "健康检查"},
			SuccessRate: 0.95,
			UsageCount:  10,
			Source:      "seed",
			UpdatedAt:   time.Now(),
		},
	}
	m.dashboardData.Categories = map[string]int{"system": 1}
	output := m.renderDashboard()
	assert.Contains(t, output, "系统健康检查")
	assert.Contains(t, output, "system")
	assert.Contains(t, output, "[内置]")
	assert.Contains(t, output, "检查系统CPU")
}

func TestRenderDashboard_SkillLearnedSource(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabSkills
	m.dashboardData.Skills = []*evolver.SkillEntry{
		{
			ID:          "learned_xxx",
			Name:        "学习技能",
			Category:    "network",
			Source:      "learned",
			SuccessRate: 0.75,
			UpdatedAt:   time.Now(),
		},
	}
	output := m.renderDashboard()
	assert.Contains(t, output, "[学习]")
}

func TestRenderDashboard_SkillManualSource(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabSkills
	m.dashboardData.Skills = []*evolver.SkillEntry{
		{
			ID:          "manual_xxx",
			Name:        "自建技能",
			Category:    "deploy",
			Source:      "manual",
			SuccessRate: 0.80,
			UpdatedAt:   time.Now(),
		},
	}
	output := m.renderDashboard()
	assert.Contains(t, output, "[自建]")
}

func TestRenderDashboard_ErrorState(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashboardData.LoadErr = "测试错误"
	output := m.renderDashboard()
	assert.Contains(t, output, "加载错误")
	assert.Contains(t, output, "测试错误")
}

func TestRenderDashboard_OverviewMemoryStats(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabOverview
	m.dashboardData.ShortTermSessions = 5
	m.dashboardData.ShortTermSizeMB = 1.23
	m.dashboardData.LongTermItems = 42
	m.dashboardData.LongTermSizeMB = 3.45
	m.dashboardData.TotalSizeMB = 4.68
	m.dashboardData.TotalLimitMB = 100
	m.dashboardData.UsagePercent = 4.68
	output := m.renderDashboard()
	assert.Contains(t, output, "5")
	assert.Contains(t, output, "4.68")
}

func TestRenderDashboard_OverviewUsageBar(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabOverview
	m.dashboardData.UsagePercent = 65.0
	m.dashboardData.TotalSizeMB = 65
	m.dashboardData.TotalLimitMB = 100
	output := m.renderDashboard()
	assert.Contains(t, output, "65.0%")
}

func TestRenderDashboard_OverviewUsageBarDanger(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabOverview
	m.dashboardData.UsagePercent = 85.0
	m.dashboardData.TotalSizeMB = 85
	m.dashboardData.TotalLimitMB = 100
	output := m.renderDashboard()
	assert.Contains(t, output, "85.0%")
}

func TestHandleDashboardKeys_Esc(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newM.(Model)
	assert.Equal(t, viewChat, model.viewState)
}

func TestHandleDashboardKeys_CtrlD(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	model := newM.(Model)
	assert.Equal(t, viewChat, model.viewState)
}

func TestHandleDashboardKeys_TabSwitch1(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	m.dashTab = tabSkills
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	model := newM.(Model)
	assert.Equal(t, tabOverview, model.dashTab)
}

func TestHandleDashboardKeys_TabSwitch2(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	m.dashTab = tabOverview
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	model := newM.(Model)
	assert.Equal(t, tabSkills, model.dashTab)
}

func TestHandleDashboardKeys_EnterToggle(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	m.dashTab = tabOverview
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := newM.(Model)
	assert.Equal(t, tabSkills, model.dashTab)
}

func TestHandleDashboardKeys_Up(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	m.dashCursor = 3
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := newM.(Model)
	assert.Equal(t, 2, model.dashCursor)
}

func TestHandleDashboardKeys_Down(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	m.dashTab = tabSkills
	m.dashboardData.Skills = []*evolver.SkillEntry{
		{ID: "a", Category: "sys"},
		{ID: "b", Category: "sys"},
		{ID: "c", Category: "sys"},
	}
	m.dashCursor = 1
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := newM.(Model)
	assert.Equal(t, 2, model.dashCursor)
}

func TestHandleDashboardKeys_DownAtMax(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	m.dashTab = tabSkills
	m.dashboardData.Skills = []*evolver.SkillEntry{
		{ID: "a", Category: "sys"},
	}
	m.dashCursor = 0
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := newM.(Model)
	assert.Equal(t, 0, model.dashCursor) // at max, should not go further
}

func TestHandleDashboardKeys_UpAtZero(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	m.dashCursor = 0
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	model := newM.(Model)
	assert.Equal(t, 0, model.dashCursor)
}

func TestHandleDashboardKeys_Refresh(t *testing.T) {
	m := createTestModelWithDashboard()
	m.viewState = viewDashboard
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	assert.NotNil(t, cmd) // should return loadDashboardCmd
	_ = newM
}

func TestRenderDashboard_OverviewSessionStats(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabOverview
	m.currentSessionID = "test-session-123"
	m.totalTokens = 500
	m.maxTokens = 4096
	m.messages = []ChatMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	output := m.renderDashboard()
	assert.Contains(t, output, "test-session-123")
	assert.Contains(t, output, "会话统计")
}

func TestRenderDashboard_OverviewNoSession(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabOverview
	m.currentSessionID = ""
	output := m.renderDashboard()
	assert.Contains(t, output, "当前会话: 无")
}

func TestRenderDashboard_SkillSuccessRateIndicators(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabSkills
	m.dashboardData.Skills = []*evolver.SkillEntry{
		{ID: "high", Name: "HighRate", Category: "sys", SuccessRate: 0.95, UpdatedAt: time.Now()},
		{ID: "med", Name: "MedRate", Category: "sys", SuccessRate: 0.85, UpdatedAt: time.Now()},
		{ID: "low", Name: "LowRate", Category: "sys", SuccessRate: 0.50, UpdatedAt: time.Now()},
	}
	output := m.renderDashboard()
	assert.Contains(t, output, "🟢")
	assert.Contains(t, output, "🟡")
	assert.Contains(t, output, "🔴")
}

func TestRenderDashboard_SkillSelectedDetail(t *testing.T) {
	m := createTestModelWithDashboard()
	m.dashTab = tabSkills
	m.dashCursor = 0
	m.dashboardData.Skills = []*evolver.SkillEntry{
		{
			ID:          "test_skill",
			Name:        "测试技能",
			Category:    "network",
			Description: "详细描述信息",
			Steps:       []string{"步骤1", "步骤2"},
			Triggers:    []string{"触发词"},
			SuccessRate: 0.90,
			UpdatedAt:   time.Now(),
		},
	}
	output := m.renderDashboard()
	assert.Contains(t, output, "详细描述信息")
	assert.Contains(t, output, "步骤1")
	assert.Contains(t, output, "触发词")
}

func TestDashboardLoadedMsg(t *testing.T) {
	data := DashboardData{
		SkillCount:    5,
		ShortTermSessions: 3,
	}
	msg := dashboardLoadedMsg{data: data}
	assert.Equal(t, 5, msg.data.SkillCount)
	assert.Equal(t, 3, msg.data.ShortTermSessions)
}

func TestCreateDashboardViewport(t *testing.T) {
	vp := createDashboardViewport(80, 24)
	assert.Equal(t, 80, vp.Width)
	assert.Equal(t, 24, vp.Height)
}

