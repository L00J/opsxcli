package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// ConfirmOption 确认选项
type ConfirmOption struct {
	Label       string
	Description string
	Value       string
}

// ConfirmDialog 确认对话框
type ConfirmDialog struct {
	Title       string
	Command     string
	CommandDesc string
	Options     []ConfirmOption
	Selected    int
	Width       int
}

// NewConfirmDialog 创建确认对话框
func NewConfirmDialog(title, command, commandDesc string) *ConfirmDialog {
	return &ConfirmDialog{
		Title:       title,
		Command:     command,
		CommandDesc: commandDesc,
		Options: []ConfirmOption{
			{
				Label:       "Yes",
				Description: "Execute this command",
				Value:       "yes",
			},
			{
				Label:       "Yes, allow similar operations",
				Description: "Execute and remember this permission",
				Value:       "yes_remember",
			},
			{
				Label:       "Type here to tell Claude what to do differently",
				Description: "Provide alternative instructions",
				Value:       "custom",
			},
		},
		Selected: 0,
		Width:    120,
	}
}

// Show 显示确认对话框
func (d *ConfirmDialog) Show() (string, error) {
	// 清屏准备显示对话框
	fmt.Println()

	// 打印分隔线
	separator := strings.Repeat("─", d.Width)
	gray := color.New(color.FgHiBlack)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	fmt.Println(separator)

	// 显示命令标题
	fmt.Printf(" %s\n\n", cyan.Sprint("Bash command"))

	// 显示命令内容
	fmt.Printf("   %s\n", d.Command)
	if d.CommandDesc != "" {
		fmt.Printf("   %s\n", gray.Sprint(d.CommandDesc))
	}
	fmt.Println()

	// 显示问题标题
	fmt.Printf(" %s\n", d.Title)

	// 显示选项
	for i, opt := range d.Options {
		prefix := "   "
		if i == d.Selected {
			prefix = yellow.Sprint(" ❯ ")
		}

		optionText := fmt.Sprintf("%d. %s", i+1, opt.Label)
		if i == d.Selected {
			fmt.Printf("%s%s\n", prefix, green.Sprint(optionText))
		} else {
			fmt.Printf("%s%s\n", prefix, optionText)
		}
	}

	fmt.Println()
	fmt.Printf(" %s\n", gray.Sprint("Esc to exit"))
	fmt.Println()

	// 读取用户输入
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)

	// 处理输入
	switch input {
	case "1":
		return "yes", nil
	case "2":
		return "yes_remember", nil
	case "3":
		fmt.Print("\n请输入您的指示: ")
		customInput, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return "custom:" + strings.TrimSpace(customInput), nil
	case "":
		// 默认选择第一个选项
		return "yes", nil
	default:
		// 如果输入的是自定义文本,作为custom处理
		return "custom:" + input, nil
	}
}

// SimpleConfirm 简单的 yes/no 确认
func SimpleConfirm(message string) bool {
	cyan := color.New(color.FgCyan)
	fmt.Printf("\n%s ", cyan.Sprint(message))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}

// ShowCommandConfirmation 显示命令确认对话框(Claude CLI 风格)
func ShowCommandConfirmation(command, description, context string) (bool, string) {
	separator := strings.Repeat("─", 120)
	gray := color.New(color.FgHiBlack)
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow)
	green := color.New(color.FgGreen)

	fmt.Println()
	fmt.Println(separator)
	fmt.Printf(" %s\n\n", cyan.Sprint("Bash command"))
	fmt.Printf("   %s\n", command)
	if description != "" {
		fmt.Printf("   %s\n", gray.Sprint(description))
	}
	fmt.Println()
	fmt.Printf(" %s\n", "Do you want to proceed?")

	// 构建选项
	options := []string{
		"Yes",
		fmt.Sprintf("Yes, allow reading from %s from this project", context),
		"Type here to tell Claude what to do differently",
	}

	// 显示选项
	for i, opt := range options {
		if i == 0 {
			fmt.Printf(" %s %d. %s\n", yellow.Sprint("❯"), i+1, green.Sprint(opt))
		} else {
			fmt.Printf("   %d. %s\n", i+1, opt)
		}
	}

	fmt.Println()
	fmt.Printf(" %s\n", gray.Sprint("Esc to exit"))
	fmt.Println()

	// 读取输入
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, ""
	}

	input = strings.TrimSpace(input)

	// 处理输入
	switch input {
	case "1", "yes", "y", "":
		return true, ""
	case "2":
		return true, "remember:" + context
	case "3":
		fmt.Print("\n请输入您的指示: ")
		customInput, err := reader.ReadString('\n')
		if err != nil {
			return false, ""
		}
		return false, strings.TrimSpace(customInput)
	default:
		// 任何其他输入作为自定义指令
		return false, input
	}
}
