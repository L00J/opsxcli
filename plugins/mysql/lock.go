package mysql

// lock.go — MySQL InnoDB 锁等待检测与死锁分析
//
// 纯函数实现，解析 SHOW ENGINE INNODB STATUS 输出中的锁等待和死锁信息。
// 不需要实际数据库连接，便于单元测试。

import (
	"fmt"
	"regexp"
	"strings"
)

// ==================== 数据结构 ====================

// InnoDBLockWait 表示一条 InnoDB 锁等待记录
type InnoDBLockWait struct {
	TransactionID int64  `json:"transaction_id"`
	ActiveSec     int64  `json:"active_seconds"`
	ThreadID      int64  `json:"thread_id"`
	Host          string `json:"host"`
	User          string `json:"user"`
	WaitSec       int64  `json:"wait_seconds"`
	WaitingQuery  string `json:"waiting_query"`
	LockMode      string `json:"lock_mode"`
	LockTable     string `json:"lock_table"`
	LockIndex     string `json:"lock_index,omitempty"`
	LockType      string `json:"lock_type,omitempty"` // RECORD, TABLE, GAP 等
}

// DeadlockParticipant 表示死锁参与事务
type DeadlockParticipant struct {
	TransactionID int64  `json:"transaction_id"`
	ActiveSec     int64  `json:"active_seconds"`
	ThreadID      int64  `json:"thread_id"`
	Host          string `json:"host,omitempty"`
	User          string `json:"user,omitempty"`
	Query         string `json:"query"`
	HoldsLock     string `json:"holds_lock,omitempty"`
	WaitingFor    string `json:"waiting_for,omitempty"`
}

// DeadlockInfo 表示检测到的死锁信息
type DeadlockInfo struct {
	Detected     bool                  `json:"detected"`
	Participants []DeadlockParticipant `json:"participants,omitempty"`
	Victims      []int64               `json:"victims,omitempty"` // 被回滚的事务 ID
}

// LockSuggestion 表示一条锁优化建议
type LockSuggestion struct {
	Level   string `json:"level"`   // info, warning, critical
	Message string `json:"message"` // 建议内容
}

// ==================== 纯函数：InnoDB 锁等待解析 ====================

// 预编译正则表达式提升性能
var (
	// reTransactionHeader 匹配事务头: ---TRANSACTION 12345, ACTIVE 30 sec ...
	reTransactionHeader = regexp.MustCompile(`---TRANSACTION\s+(\d+),\s+ACTIVE\s+(\d+)\s+sec`)
	// 匹配死锁区域中的事务头（没有 --- 前缀）: TRANSACTION 12345, ACTIVE 30 sec
	reTransactionHeaderNoDash = regexp.MustCompile(`TRANSACTION\s+(\d+),\s+ACTIVE\s+(\d+)\s+sec`)
	// 匹配 LOCK WAIT 标记
	reLockWait = regexp.MustCompile(`LOCK WAIT`)
	// 匹配死锁区域事务: *** (1) TRANSACTION:
	reDeadlockTx = regexp.MustCompile(`\*{3}\s*\(\d+\)\s*TRANSACTION:`)
	// 匹配死锁区域标题
	reDeadlockHeader = regexp.MustCompile(`LATEST DETECTED DEADLOCK`)
	// 匹配回滚事务: WE ROLL BACK TRANSACTION (2)
	reRollback = regexp.MustCompile(`WE ROLL BACK TRANSACTION\s+\((\d+)\)`)
	// 匹配 MySQL 线程信息: MySQL thread id 10, OS thread handle 0x1234, query id 100 localhost root updating
	reMySQLThread = regexp.MustCompile(`MySQL thread id\s+(\d+),\s+OS thread handle\s+\S+,\s+query id\s+\d+\s+(\S+)\s+(\S+)\s+(\S+)`)
	// 匹配等待时间: TRX HAS BEEN WAITING 5 SEC
	reWaitSec = regexp.MustCompile(`TRX HAS BEEN WAITING\s+(\d+)\s+SEC`)
	// 匹配锁模式: lock_mode X ...
	reLockMode = regexp.MustCompile(`lock_mode\s+(\S+)`)
	// 匹配表名: of table `db`.`table` 或 table `db`.`table`
	reTableName = regexp.MustCompile("of table\\s+`([^`]+)`.`([^`]+)`")
	// 匹配索引名: index `idx_name`
	reIndexName = regexp.MustCompile("index\\s+`([^`]+)`")
)

// ParseInnoDBLockWaits 解析 SHOW ENGINE INNODB STATUS 输出中的锁等待信息
func ParseInnoDBLockWaits(innodbStatus string) []InnoDBLockWait {
	if innodbStatus == "" {
		return nil
	}

	var locks []InnoDBLockWait

	// 解析常规锁等待区域（TRANSACTIONS section 中的 LOCK WAIT 事务）
	locks = append(locks, parseLockWaitsFromSection(innodbStatus)...)

	// 解析死锁区域中的锁等待
	locks = append(locks, parseDeadlockLockWaits(innodbStatus)...)

	return locks
}

// parseDeadlockLockWaits 从死锁区域中提取锁等待信息
func parseDeadlockLockWaits(text string) []InnoDBLockWait {
	// 检查是否存在死锁区域
	if !reDeadlockHeader.MatchString(text) {
		return nil
	}

	deadlockIdx := reDeadlockHeader.FindStringIndex(text)
	if deadlockIdx == nil {
		return nil
	}
	deadlockSection := text[deadlockIdx[0]:]

	var locks []InnoDBLockWait

	// 按 *** (N) TRANSACTION: 拆分（使用预编译正则）
	txPositions := reDeadlockTx.FindAllStringIndex(deadlockSection, -1)

	for i, pos := range txPositions {
		start := pos[0]
		end := len(deadlockSection)
		if i+1 < len(txPositions) {
			end = txPositions[i+1][0]
		}

		section := deadlockSection[start:end]

		// 只处理包含 WAITING FOR 的死锁事务
		if !strings.Contains(section, "WAITING FOR THIS LOCK") && !strings.Contains(section, "LOCK WAIT") {
			// 也检查是否包含 WAITING FOR
			if !strings.Contains(section, "WAITING FOR") {
				continue
			}
		}

		lw := InnoDBLockWait{}

		// 解析事务 ID 和活跃时间（死锁区域格式无 --- 前缀）
		txMatch := reTransactionHeaderNoDash.FindStringSubmatch(section)
		if len(txMatch) >= 3 {
			parseInt64(txMatch[1], &lw.TransactionID)
			parseInt64(txMatch[2], &lw.ActiveSec)
		}

		// 解析线程信息
		threadMatch := reMySQLThread.FindStringSubmatch(section)
		if len(threadMatch) >= 5 {
			parseInt64(threadMatch[1], &lw.ThreadID)
			lw.Host = threadMatch[2]
			lw.User = threadMatch[3]
		}

		// 提取等待的 SQL
		lines := strings.Split(section, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "***") ||
				strings.HasPrefix(line, "---") ||
				strings.Contains(line, "MySQL thread id") ||
				strings.Contains(line, "TRANSACTION") ||
				strings.Contains(line, "LOCK WAIT") ||
				strings.Contains(line, "WAITING FOR") ||
				strings.Contains(line, "HOLDS THE LOCK") ||
				strings.Contains(line, "RECORD LOCKS") ||
				strings.Contains(line, "Record lock") ||
				strings.HasPrefix(line, "0:") ||
				strings.HasPrefix(line, "1:") ||
				strings.HasPrefix(line, "2:") ||
				strings.HasPrefix(line, "mysql tables") {
				continue
			}
			if isSQLLike(line) {
				lw.WaitingQuery = line
				break
			}
		}

		// 解析锁模式
		lockMatch := reLockMode.FindStringSubmatch(section)
		if len(lockMatch) >= 2 {
			lw.LockMode = lockMatch[1]
		}

		// 解析表名
		tableMatch := reTableName.FindStringSubmatch(section)
		if len(tableMatch) >= 3 {
			lw.LockTable = tableMatch[1] + "." + tableMatch[2]
		}

		// 解析索引名
		indexMatch := reIndexName.FindStringSubmatch(section)
		if len(indexMatch) >= 2 {
			lw.LockIndex = indexMatch[1]
		}

		// 死锁中的等待时间通常不单独列出，默认为0
		lw.WaitSec = 0

		locks = append(locks, lw)
	}

	return locks
}

// parseLockWaitsFromSection 从 INNODB STATUS 文本中解析所有锁等待
func parseLockWaitsFromSection(text string) []InnoDBLockWait {
	var locks []InnoDBLockWait

	// 按事务头拆分
	matches := reTransactionHeader.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	for i, match := range matches {
		// 从事务头开始到下一个事务头或文本结尾
		start := match[0]
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		section := text[start:end]

		// 只处理包含 LOCK WAIT 的事务
		if !reLockWait.MatchString(section) {
			continue
		}

		lw := InnoDBLockWait{}

		// 解析事务 ID 和活跃时间
		txMatch := reTransactionHeader.FindStringSubmatch(section)
		if len(txMatch) >= 3 {
			parseInt64(txMatch[1], &lw.TransactionID)
			parseInt64(txMatch[2], &lw.ActiveSec)
		}

		// 解析线程信息
		threadMatch := reMySQLThread.FindStringSubmatch(section)
		if len(threadMatch) >= 5 {
			parseInt64(threadMatch[1], &lw.ThreadID)
			lw.Host = threadMatch[2]
			lw.User = threadMatch[3]
		}

		// 解析等待时间
		waitMatch := reWaitSec.FindStringSubmatch(section)
		if len(waitMatch) >= 2 {
			parseInt64(waitMatch[1], &lw.WaitSec)
		}

		// 提取等待的 SQL（事务头之后的非空行）
		lines := strings.Split(section, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// 跳过以 --- 或特定关键词开头的行
			if line == "" || strings.HasPrefix(line, "---") ||
				strings.HasPrefix(line, "mysql tables") ||
				strings.HasPrefix(line, "LOCK WAIT") ||
				strings.Contains(line, "MySQL thread id") ||
				strings.Contains(line, "TRX HAS BEEN WAITING") ||
				strings.Contains(line, "RECORD LOCKS") ||
				strings.Contains(line, "Record lock") ||
				strings.HasPrefix(line, "0:") ||
				strings.HasPrefix(line, "1:") ||
				strings.HasPrefix(line, "2:") ||
				strings.HasPrefix(line, "***") {
				continue
			}
			// 这行可能是 SQL
			if isSQLLike(line) {
				lw.WaitingQuery = line
				break
			}
		}

		// 解析锁模式
		lockMatch := reLockMode.FindStringSubmatch(section)
		if len(lockMatch) >= 2 {
			lw.LockMode = lockMatch[1]
		}

		// 解析表名
		tableMatch := reTableName.FindStringSubmatch(section)
		if len(tableMatch) >= 3 {
			lw.LockTable = tableMatch[1] + "." + tableMatch[2]
		}

		// 解析索引名
		indexMatch := reIndexName.FindStringSubmatch(section)
		if len(indexMatch) >= 2 {
			lw.LockIndex = indexMatch[1]
		}

		locks = append(locks, lw)
	}

	return locks
}

// parseInt64 安全解析 int64
func parseInt64(s string, target *int64) {
	if s == "" {
		return
	}
	// 手动解析避免 strconv 依赖
	var n int64
	neg := false
	for i, c := range s {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	*target = n
}

// isSQLLike 判断一行文本是否像 SQL 语句
func isSQLLike(line string) bool {
	upper := strings.ToUpper(line)
	sqlPrefixes := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "REPLACE", "CREATE", "ALTER", "DROP", "SHOW", "EXPLAIN"}
	for _, p := range sqlPrefixes {
		if strings.HasPrefix(upper, p) {
			return true
		}
	}
	return false
}

// ==================== 纯函数：死锁检测 ====================

// DetectDeadlocks 从 INNODB STATUS 中检测死锁信息
func DetectDeadlocks(innodbStatus string) DeadlockInfo {
	if innodbStatus == "" {
		return DeadlockInfo{}
	}

	// 检查是否存在死锁区域
	if !reDeadlockHeader.MatchString(innodbStatus) {
		return DeadlockInfo{}
	}

	info := DeadlockInfo{Detected: true}

	// 提取死锁区域（从 LATEST DETECTED DEADLOCK 到下一个分隔线或 END）
	deadlockIdx := reDeadlockHeader.FindStringIndex(innodbStatus)
	if deadlockIdx == nil {
		return info
	}

	deadlockSection := innodbStatus[deadlockIdx[0]:]

	// 解析参与事务 (1) 和 (2)
	// 按 *** (N) TRANSACTION: 拆分（使用预编译正则）
	txMatches := reDeadlockTx.FindAllStringSubmatchIndex(deadlockSection, -1)

	for i, match := range txMatches {
		start := match[0]
		end := len(deadlockSection)
		if i+1 < len(txMatches) {
			end = txMatches[i+1][0]
		}

		section := deadlockSection[start:end]

		p := DeadlockParticipant{}

		// 解析事务 ID（死锁区域格式无 --- 前缀）
		txMatch := reTransactionHeaderNoDash.FindStringSubmatch(section)
		if len(txMatch) >= 3 {
			parseInt64(txMatch[1], &p.TransactionID)
			parseInt64(txMatch[2], &p.ActiveSec)
		}

		// 解析线程信息
		threadMatch := reMySQLThread.FindStringSubmatch(section)
		if len(threadMatch) >= 5 {
			parseInt64(threadMatch[1], &p.ThreadID)
			p.Host = threadMatch[2]
			p.User = threadMatch[3]
		}

		// 提取 SQL
		lines := strings.Split(section, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if isSQLLike(line) {
				p.Query = line
				break
			}
		}

		// 解析持有的锁
		holdsMatch := regexp.MustCompile(`HOLDS THE LOCK\(S\):([\s\S]*?)(?:\*{3}|$)`).FindStringSubmatch(section)
		if len(holdsMatch) >= 2 {
			lockParts := reLockMode.FindStringSubmatch(holdsMatch[1])
			if len(lockParts) >= 2 {
				p.HoldsLock = lockParts[1]
			}
		}

		// 解析等待的锁
		waitSection := section
		waitIdx := strings.Index(section, "WAITING FOR THIS LOCK")
		if waitIdx >= 0 {
			waitSection = section[waitIdx:]
		}
		lockMatch := reLockMode.FindStringSubmatch(waitSection)
		if len(lockMatch) >= 2 {
			p.WaitingFor = lockMatch[1]
		}

		info.Participants = append(info.Participants, p)
	}

	// 解析被回滚的事务
	rollbackMatches := reRollback.FindAllStringSubmatch(deadlockSection, -1)
	for _, m := range rollbackMatches {
		if len(m) >= 2 {
			var victimID int64
			parseInt64(m[1], &victimID)
			// 回滚标记对应的是组号 (1)/(2)，需要映射到实际事务 ID
			// WE ROLL BACK TRANSACTION (N) 中的 N 是组号
			groupIdx := int(victimID) - 1 // 0-indexed
			if groupIdx >= 0 && groupIdx < len(info.Participants) {
				info.Victims = append(info.Victims, info.Participants[groupIdx].TransactionID)
			}
		}
	}

	return info
}

// ==================== 纯函数：锁模式分类 ====================

// CategorizeLockMode 将 InnoDB 锁模式字符串分类为中文描述
func CategorizeLockMode(lockStr string) string {
	if lockStr == "" {
		return "未知"
	}

	upper := strings.ToUpper(lockStr)

	// 提取锁类型字母
	lockType := ""
	if idx := strings.Index(upper, "LOCK_MODE"); idx >= 0 {
		rest := strings.TrimSpace(upper[idx+len("lock_mode"):])
		if rest != "" {
			// 提取第一个词
			parts := strings.SplitN(rest, " ", 2)
			lockType = parts[0]
		}
	} else {
		// 直接传入的可能是 "X locks rec but not gap" 格式
		parts := strings.SplitN(upper, " ", 2)
		lockType = parts[0]
	}

	switch lockType {
	case "X":
		if strings.Contains(upper, "GAP BEFORE REC") {
			return "排他间隙锁(X)"
		}
		return "排他锁(X)"
	case "S":
		if strings.Contains(upper, "GAP BEFORE REC") {
			return "共享间隙锁(S)"
		}
		return "共享锁(S)"
	case "IX":
		return "意向排他锁(IX)"
	case "IS":
		return "意向共享锁(IS)"
	case "AUTO_INC":
		return "自动增量锁"
	default:
		if lockType != "" {
			return "其他(" + lockType + ")"
		}
		return "未知"
	}
}

// ==================== 纯函数：锁严重度分析 ====================

// AnalyzeLockSeverity 分析锁等待的严重程度
func AnalyzeLockSeverity(waits []InnoDBLockWait, deadlock DeadlockInfo) string {
	// 死锁始终是严重级别
	if deadlock.Detected {
		return "严重"
	}

	if len(waits) == 0 {
		return "正常"
	}

	// 找到最大等待时间
	var maxWait int64
	for _, w := range waits {
		if w.WaitSec > maxWait {
			maxWait = w.WaitSec
		}
	}

	switch {
	case maxWait >= 30:
		return "高风险"
	case maxWait >= 5:
		return "中风险"
	default:
		return "低风险"
	}
}

// ==================== 纯函数：锁等待建议生成 ====================

// GenerateLockWaitSuggestions 根据锁等待信息生成优化建议
func GenerateLockWaitSuggestions(waits []InnoDBLockWait, deadlock DeadlockInfo) []LockSuggestion {
	var suggestions []LockSuggestion

	// 死锁建议
	if deadlock.Detected {
		suggestions = append(suggestions, LockSuggestion{
			Level:   "critical",
			Message: fmt.Sprintf("检测到死锁！涉及 %d 个事务，被回滚事务ID: %v。建议检查事务访问表的顺序是否一致", len(deadlock.Participants), deadlock.Victims),
		})
	}

	if len(waits) == 0 {
		return suggestions
	}

	// 分析等待时间
	var maxWait int64
	var totalWait int64
	for _, w := range waits {
		totalWait += w.WaitSec
		if w.WaitSec > maxWait {
			maxWait = w.WaitSec
		}
	}

	// 长等待建议
	if maxWait >= 30 {
		suggestions = append(suggestions, LockSuggestion{
			Level:   "critical",
			Message: fmt.Sprintf("最长锁等待 %d 秒，可能导致业务超时。建议优化大事务、拆分批量操作", maxWait),
		})
	} else if maxWait >= 10 {
		suggestions = append(suggestions, LockSuggestion{
			Level:   "warning",
			Message: fmt.Sprintf("锁等待 %d 秒，建议检查是否缺少索引导致锁范围过大", maxWait),
		})
	}

	// 多条等待建议
	if len(waits) >= 3 {
		suggestions = append(suggestions, LockSuggestion{
			Level:   "warning",
			Message: fmt.Sprintf("当前 %d 个锁等待，总等待 %d 秒。建议检查热点表或行竞争", len(waits), totalWait),
		})
	}

	return suggestions
}

// ==================== 纯函数：报告格式化 ====================

// FormatLockWaitReport 格式化锁等待分析报告
func FormatLockWaitReport(waits []InnoDBLockWait, deadlock DeadlockInfo) string {
	if len(waits) == 0 && !deadlock.Detected {
		return "未检测到锁等待"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== 锁等待分析报告 (%d 条等待) ===\n\n", len(waits)))

	// 严重度
	severity := AnalyzeLockSeverity(waits, deadlock)
	b.WriteString(fmt.Sprintf("严重程度: %s\n\n", severity))

	// 锁等待列表
	if len(waits) > 0 {
		b.WriteString("--- 锁等待列表 ---\n")
		for i, w := range waits {
			b.WriteString(fmt.Sprintf("[%d] 事务ID: %d | 等待: %ds | 活跃: %ds\n",
				i+1, w.TransactionID, w.WaitSec, w.ActiveSec))
			b.WriteString(fmt.Sprintf("    用户: %s@%s (线程 %d)\n", w.User, w.Host, w.ThreadID))

			sqlDisplay := w.WaitingQuery
			if len(sqlDisplay) > 100 {
				sqlDisplay = sqlDisplay[:100] + "..."
			}
			b.WriteString(fmt.Sprintf("    SQL: %s\n", sqlDisplay))

			if w.LockMode != "" {
				b.WriteString(fmt.Sprintf("    锁模式: %s | 表: %s", w.LockMode, w.LockTable))
				if w.LockIndex != "" {
					b.WriteString(fmt.Sprintf(" | 索引: %s", w.LockIndex))
				}
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// 死锁信息
	if deadlock.Detected {
		b.WriteString("⚠️ 检测到死锁\n\n")
		if len(deadlock.Participants) > 0 {
			b.WriteString("--- 死锁参与事务 ---\n")
			for i, p := range deadlock.Participants {
				b.WriteString(fmt.Sprintf("  (%d) 事务ID: %d | 线程: %d\n", i+1, p.TransactionID, p.ThreadID))
				sqlDisplay := p.Query
				if len(sqlDisplay) > 80 {
					sqlDisplay = sqlDisplay[:80] + "..."
				}
				b.WriteString(fmt.Sprintf("      SQL: %s\n", sqlDisplay))
			}
		}
		if len(deadlock.Victims) > 0 {
			b.WriteString(fmt.Sprintf("\n被回滚事务ID: %v\n", deadlock.Victims))
		}
		b.WriteString("\n")
	}

	// 优化建议
	suggestions := GenerateLockWaitSuggestions(waits, deadlock)
	if len(suggestions) > 0 {
		b.WriteString("--- 优化建议 ---\n")
		for _, s := range suggestions {
			icon := "💡"
			if s.Level == "critical" {
				icon = "🔴"
			} else if s.Level == "warning" {
				icon = "🟡"
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", icon, s.Message))
		}
	}

	return b.String()
}
