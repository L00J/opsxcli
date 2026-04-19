package tui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var (
	inlineCodeRe = regexp.MustCompile("`([^`]+)`")
	boldRe       = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	ansiRe       = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

// RenderMarkdown 将 Markdown 文本渲染为带 ANSI 样式的字符串
// 支持：标题、代码块、行内代码、列表、粗体、引用、表格、分隔线
func RenderMarkdown(text string, width int) string {
	if width < 10 {
		width = 10
	}

	lines := strings.Split(text, "\n")
	var blocks []string
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 代码块
		if strings.HasPrefix(trimmed, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			i++
			start := i
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				i++
			}
			code := strings.Join(lines[start:i], "\n")
			blocks = append(blocks, renderCodeBlock(code, lang))
			if i < len(lines) {
				i++ // skip closing ```
			}
			continue
		}

		// 分隔线
		if isDivider(trimmed) {
			divWidth := width - 4
			if divWidth < 4 {
				divWidth = 4
			}
			blocks = append(blocks, dividerStyle.Render(strings.Repeat("─", divWidth)))
			i++
			continue
		}

		// 标题
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			content := processInline(strings.TrimPrefix(trimmed, "# "))
			blocks = append(blocks, h1Style.Render(content))
			i++
			continue
		}
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			content := processInline(strings.TrimPrefix(trimmed, "## "))
			blocks = append(blocks, h2Style.Render(content))
			i++
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			content := processInline(strings.TrimPrefix(trimmed, "### "))
			blocks = append(blocks, h3Style.Render(content))
			i++
			continue
		}

		// 引用
		if strings.HasPrefix(trimmed, ">") {
			var quoteLines []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				qline := strings.TrimSpace(lines[i])
				qline = strings.TrimPrefix(qline, ">")
				qline = strings.TrimPrefix(qline, " ")
				quoteLines = append(quoteLines, qline)
				i++
			}
			qcontent := processInline(strings.Join(quoteLines, " "))
			blocks = append(blocks, quoteStyle.Render(qcontent))
			continue
		}

		// 表格
		if isTableLine(trimmed) {
			var tableLines []string
			for i < len(lines) && isTableLine(strings.TrimSpace(lines[i])) {
				tableLines = append(tableLines, lines[i])
				i++
			}
			// 跳过分隔行 |---|---|
			if len(tableLines) >= 2 && isTableSeparator(tableLines[1]) {
				tableLines = append(tableLines[:1], tableLines[2:]...)
			}
			blocks = append(blocks, renderTable(tableLines, width))
			continue
		}

		// 列表
		if isListItem(trimmed) {
			var listItems []string
			for i < len(lines) {
				currTrimmed := strings.TrimSpace(lines[i])
				if isListItem(currTrimmed) {
					marker, text := parseListItem(currTrimmed)
					rendered := listStyle.Render(marker + " " + processInline(text))
					listItems = append(listItems, "  "+rendered)
					i++
				} else if currTrimmed == "" {
					// 检查下一行是否是列表项或新块
					if i+1 < len(lines) && !isBlockStart(lines[i+1]) && isListItem(strings.TrimSpace(lines[i+1])) {
						i++
						continue
					}
					break
				} else if isBlockStart(lines[i]) {
					break
				} else {
					// 列表项续行
					rendered := listStyle.Render("  " + processInline(currTrimmed))
					listItems = append(listItems, "    "+rendered)
					i++
				}
			}
			blocks = append(blocks, strings.Join(listItems, "\n"))
			continue
		}

		// 空行
		if trimmed == "" {
			i++
			continue
		}

		// 普通段落
		var paraLines []string
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == "" {
				break
			}
			if isBlockStart(lines[i]) {
				break
			}
			paraLines = append(paraLines, lines[i])
			i++
		}
		para := strings.Join(paraLines, " ")
		blocks = append(blocks, processInline(para))
	}

	return strings.Join(blocks, "\n\n")
}

// isDivider 判断是否为分隔线
func isDivider(s string) bool {
	if len(s) < 3 {
		return false
	}
	for _, c := range s {
		if c != '-' && c != '*' && c != '_' && c != ' ' && c != '\t' {
			return false
		}
	}
	return strings.Contains(s, "---") || strings.Contains(s, "***") || strings.Contains(s, "___")
}

// isBlockStart 判断是否为新块的开始
func isBlockStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "```") ||
		strings.HasPrefix(trimmed, ">") ||
		isDivider(trimmed) ||
		isTableLine(trimmed) ||
		isListItem(trimmed)
}

// isTableLine 判断是否为表格行
func isTableLine(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "|") && strings.HasSuffix(s, "|") && strings.Count(s, "|") >= 2
}

// isTableSeparator 判断是否为表格分隔行
func isTableSeparator(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "|") {
		return false
	}
	parts := strings.Split(s, "|")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		for _, c := range p {
			if c != '-' && c != ':' && c != ' ' && c != '\t' {
				return false
			}
		}
	}
	return true
}

// isListItem 判断是否为列表项
func isListItem(s string) bool {
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") {
		return true
	}
	if len(s) >= 3 {
		dotIdx := strings.Index(s, ". ")
		if dotIdx > 0 {
			prefix := s[:dotIdx]
			isNum := true
			for _, c := range prefix {
				if c < '0' || c > '9' {
					isNum = false
					break
				}
			}
			if isNum && len(prefix) <= 3 {
				return true
			}
		}
	}
	return false
}

// parseListItem 解析列表项，返回标记符号和文本
func parseListItem(s string) (string, string) {
	if strings.HasPrefix(s, "- ") {
		return "•", strings.TrimPrefix(s, "- ")
	}
	if strings.HasPrefix(s, "* ") {
		return "•", strings.TrimPrefix(s, "* ")
	}
	dotIdx := strings.Index(s, ". ")
	if dotIdx > 0 {
		return s[:dotIdx+1], strings.TrimPrefix(s[dotIdx:], ". ")
	}
	return "•", s
}

// renderCodeBlock 渲染代码块
func renderCodeBlock(code, lang string) string {
	// 基础高亮
	highlighted := highlightCode(code, lang)

	// 语言标签
	var header string
	if lang != "" {
		header = codeBlockLangStyle.Render(lang) + "\n"
	}

	// 代码内容
	codeRendered := codeBlockStyle.Render(highlighted)
	if header != "" {
		return header + codeRendered
	}
	return codeRendered
}

// highlightCode 对代码进行基础语法高亮
func highlightCode(code, lang string) string {
	switch strings.ToLower(lang) {
	case "bash", "sh", "shell", "zsh":
		return highlightBash(code)
	case "json":
		return highlightJSON(code)
	case "go", "golang":
		return highlightGo(code)
	default:
		return highlightGeneric(code)
	}
}

// highlightBash bash 高亮
func highlightBash(code string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 注释
		if strings.HasPrefix(trimmed, "#") {
			lines[i] = bashCommentStyle.Render(line)
			continue
		}
		// $ 开头的命令行
		if strings.HasPrefix(trimmed, "$") {
			parts := strings.SplitN(trimmed, " ", 2)
			if len(parts) == 2 {
				prompt := bashPromptStyle.Render(parts[0])
				cmd := bashCommandStyle.Render(parts[1])
				lines[i] = prompt + " " + cmd
			} else {
				lines[i] = bashPromptStyle.Render(line)
			}
			continue
		}
		// 关键字
		lines[i] = highlightKeywords(line, []string{"if", "then", "else", "fi", "for", "do", "done", "while", "case", "esac", "echo", "export", "return"})
	}
	return strings.Join(lines, "\n")
}

// highlightJSON JSON 高亮
func highlightJSON(code string) string {
	// 字符串
	code = highlightPattern(code, `"([^"\\]|\\.)*"`, bashStringStyle)
	// 数字
	code = highlightPattern(code, `\b\d+(\.\d+)?\b`, bashNumberStyle)
	// 关键字 (true, false, null)
	code = highlightPattern(code, `\b(true|false|null)\b`, bashKeywordStyle)
	return code
}

// highlightGo Go 高亮
func highlightGo(code string) string {
	// 字符串
	code = highlightPattern(code, "`[^`]*`", bashStringStyle)
	code = highlightPattern(code, `"([^"\\]|\\.)*"`, bashStringStyle)
	// 注释
	code = highlightPattern(code, `//.*$`, bashCommentStyle)
	// 数字
	code = highlightPattern(code, `\b\d+(\.\d+)?\b`, bashNumberStyle)
	// 关键字
	keywords := []string{"package", "import", "func", "return", "var", "const", "type", "struct", "interface", "if", "else", "for", "range", "go", "defer", "switch", "case", "default", "break", "continue", "map", "chan"}
	code = highlightKeywords(code, keywords)
	return code
}

// highlightGeneric 通用高亮
func highlightGeneric(code string) string {
	// 注释
	code = highlightPattern(code, `#.*$`, bashCommentStyle)
	// 字符串
	code = highlightPattern(code, `"([^"\\]|\\.)*"`, bashStringStyle)
	// 数字
	code = highlightPattern(code, `\b\d+(\.\d+)?\b`, bashNumberStyle)
	return code
}

// highlightKeywords 高亮关键字
func highlightKeywords(code string, keywords []string) string {
	for _, kw := range keywords {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(kw) + `\b`)
		code = re.ReplaceAllString(code, bashKeywordStyle.Render(kw))
	}
	return code
}

// highlightPattern 使用正则高亮匹配的模式
func highlightPattern(code, pattern string, style lipgloss.Style) string {
	re := regexp.MustCompile("(?m)" + pattern)
	return re.ReplaceAllStringFunc(code, func(match string) string {
		return style.Render(match)
	})
}

// processInline 处理行内元素（粗体、行内代码）
func processInline(text string) string {
	// 先处理行内代码，再处理粗体
	text = processInlineCode(text)
	text = processBold(text)
	return text
}

func processInlineCode(text string) string {
	return inlineCodeRe.ReplaceAllString(text, inlineCodeStyle.Render("$1"))
}

func processBold(text string) string {
	return boldRe.ReplaceAllString(text, boldStyle.Render("$1"))
}

// renderTable 渲染简单表格
func renderTable(lines []string, width int) string {
	if len(lines) == 0 {
		return ""
	}

	// 解析所有单元格
	var rows [][]string
	for _, line := range lines {
		cells := parseTableCells(line)
		rows = append(rows, cells)
	}

	// 计算列数
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	if maxCols == 0 {
		return ""
	}

	// 计算每列最大宽度
	colWidths := make([]int, maxCols)
	for _, row := range rows {
		for i, cell := range row {
			w := utf8.RuneCountInString(stripANSI(cell))
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	// 分配剩余宽度
	totalWidth := 0
	for _, w := range colWidths {
		totalWidth += w
	}
	totalWidth += (maxCols + 1) * 3 // borders and padding

	// 如果总宽超过限制，等比例缩小
	if totalWidth > width && totalWidth > 0 {
		ratio := float64(width-((maxCols+1)*3)) / float64(totalWidth-((maxCols+1)*3))
		for i := range colWidths {
			colWidths[i] = int(float64(colWidths[i]) * ratio)
			if colWidths[i] < 3 {
				colWidths[i] = 3
			}
		}
	}

	var b strings.Builder
	for _, row := range rows {
		b.WriteString("│ ")
		for i, cell := range row {
			if i > 0 {
				b.WriteString(" │ ")
			}
			cellWidth := utf8.RuneCountInString(stripANSI(cell))
			padding := colWidths[i] - cellWidth
			if padding < 0 {
				padding = 0
			}
			b.WriteString(cell)
			b.WriteString(strings.Repeat(" ", padding))
		}
		b.WriteString(" │\n")
	}

	tableStr := strings.TrimSuffix(b.String(), "\n")
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│", TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘"}).
		BorderForeground(lipgloss.Color("#585b70")).
		Padding(0, 1).
		Render(tableStr)
}

// parseTableCells 解析表格行单元格
func parseTableCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// stripANSI 去除 ANSI 转义序列
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
