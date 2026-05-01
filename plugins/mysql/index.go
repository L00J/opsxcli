package mysql

// index.go — MySQL 索引分析与建议
//
// 提供 SHOW INDEX 输出解析、冗余索引检测、复合索引建议等纯函数。
// 不需要实际数据库连接，便于单元测试。

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ==================== 数据结构 ====================

// ShowIndexEntry 表示 SHOW INDEX FROM table 输出的一条记录
type ShowIndexEntry struct {
	Table       string `json:"table"`
	NonUnique   bool   `json:"non_unique"`
	KeyName     string `json:"key_name"`
	SeqInIndex  int    `json:"seq_in_index"`
	ColumnName  string `json:"column_name"`
	Collation   string `json:"collation"`
	Cardinality int64  `json:"cardinality"`
	SubPart     int    `json:"sub_part"`
	Nullable    bool   `json:"nullable"`
	IndexType   string `json:"index_type"`
	Expression  string `json:"expression,omitempty"`
}

// RedundantIndexInfo 表示一个冗余索引的信息
type RedundantIndexInfo struct {
	Table          string   `json:"table"`
	RedundantIndex string   `json:"redundant_index"`
	CoveredBy      string   `json:"covered_by"`
	WastedColumns  []string `json:"wasted_columns"`
}

// CompositeIndexSuggestion 表示复合索引建议
type CompositeIndexSuggestion struct {
	Table            string   `json:"table"`
	SuggestedColumns []string `json:"suggested_columns"`
	Reason           string   `json:"reason"`
	Confidence       float64  `json:"confidence"`
	SQL              string   `json:"suggested_sql"`
}

// ==================== 纯函数：SHOW INDEX 解析 ====================

// 预编译正则：提取 WHERE 子句中的列名
var (
	reWhereColumns  = regexp.MustCompile(`(?i)\bWHERE\s+(.+?)(?:\s+GROUP\s+BY|\s+ORDER\s+BY|\s+LIMIT|\s+HAVING|$)`)
	reColumnRef     = regexp.MustCompile(`(?i)(?:^|[\s(])` + "`" + `?(\w+)` + "`" + `?\s*(?:=|!=|<>|>|<|>=|<=|\s+IN\s*\(|\s+BETWEEN\s+|\s+LIKE\s+|\s+IS\s+)`)
	reFromTableName = regexp.MustCompile(`(?i)\bFROM\s+` + "`" + `?(\w+)` + "`" + `?`)
)

// ParseShowIndexOutput 解析 SHOW INDEX FROM table 的 tab 分隔输出
// 输入是 MySQL 命令行客户端的原始输出（TSV 格式）
// 第一行是标题行（以 Table 开头），会自动跳过
func ParseShowIndexOutput(output string) []ShowIndexEntry {
	if output == "" {
		return nil
	}

	lines := strings.Split(output, "\n")
	var entries []ShowIndexEntry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 跳过标题行（以 "Table" 开头）
		if strings.HasPrefix(line, "Table\t") {
			continue
		}

		entry, ok := ParseShowIndexRow(line)
		if ok {
			entries = append(entries, entry)
		}
	}

	return entries
}

// ParseShowIndexRow 解析 SHOW INDEX 输出的单行（tab 分隔）
// SHOW INDEX 输出列：Table, Non_unique, Key_name, Seq_in_index, Column_name,
//
//	Collation, Cardinality, Sub_part, Packed, Null, Index_type, Comment, Index_comment, Visible, Expression
func ParseShowIndexRow(line string) (ShowIndexEntry, bool) {
	fields := strings.Split(line, "\t")
	// 至少需要 7 个字段：Table, Non_unique, Key_name, Seq_in_index, Column_name, Collation, Cardinality
	if len(fields) < 7 {
		return ShowIndexEntry{}, false
	}

	entry := ShowIndexEntry{
		Table:      strings.TrimSpace(fields[0]),
		KeyName:    strings.TrimSpace(fields[2]),
		ColumnName: strings.TrimSpace(fields[4]),
		Collation:  strings.TrimSpace(fields[5]),
		IndexType:  "BTREE",
	}

	// Non_unique: 0=唯一, 1=非唯一
	if fields[1] == "0" {
		entry.NonUnique = false
	} else {
		entry.NonUnique = true
	}

	// Seq_in_index
	parseSeqInIndex(fields[3], &entry)

	// Cardinality（可能为 NULL）
	card := strings.TrimSpace(fields[6])
	if card != "NULL" && card != "" {
		var n int64
		parseInt64(card, &n)
		entry.Cardinality = n
	}

	// Sub_part（可选字段，索引前缀长度）
	if len(fields) >= 8 {
		sub := strings.TrimSpace(fields[7])
		if sub != "NULL" && sub != "" {
			var n int64
			parseInt64(sub, &n)
			entry.SubPart = int(n)
		}
	}

	// Nullable
	if len(fields) >= 10 {
		entry.Nullable = strings.TrimSpace(fields[9]) == "YES"
	}

	// Index_type
	if len(fields) >= 11 {
		entry.IndexType = strings.TrimSpace(fields[10])
	}

	// Expression（MySQL 8.0+）
	if len(fields) >= 15 {
		expr := strings.TrimSpace(fields[14])
		if expr != "NULL" && expr != "" {
			entry.Expression = expr
		}
	}

	return entry, true
}

// parseSeqInIndex 解析 Seq_in_index 字段
func parseSeqInIndex(s string, entry *ShowIndexEntry) {
	s = strings.TrimSpace(s)
	if s == "" || s == "NULL" {
		return
	}
	var n int64
	parseInt64(s, &n)
	entry.SeqInIndex = int(n)
}

// ==================== 纯函数：索引分组 ====================

// GroupIndexesByTable 按表名分组索引条目
func GroupIndexesByTable(entries []ShowIndexEntry) map[string][]ShowIndexEntry {
	result := make(map[string][]ShowIndexEntry)
	for _, e := range entries {
		result[e.Table] = append(result[e.Table], e)
	}
	return result
}

// BuildIndexColumnsMap 将同一表的索引条目构建为 {索引名: [列名列表]} 映射
// 列名按 SeqInIndex 排序
func BuildIndexColumnsMap(entries []ShowIndexEntry) map[string][]string {
	// 先按 SeqInIndex 排序
	sorted := make([]ShowIndexEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].KeyName != sorted[j].KeyName {
			return sorted[i].KeyName < sorted[j].KeyName
		}
		return sorted[i].SeqInIndex < sorted[j].SeqInIndex
	})

	result := make(map[string][]string)
	for _, e := range sorted {
		result[e.KeyName] = append(result[e.KeyName], e.ColumnName)
	}
	return result
}

// ==================== 纯函数：冗余索引检测 ====================

// DetectRedundantIndexes 检测冗余索引
// 索引 B 的列是索引 A 的前缀时，B 被 A 覆盖，是冗余的
func DetectRedundantIndexes(entries []ShowIndexEntry) []RedundantIndexInfo {
	grouped := GroupIndexesByTable(entries)
	var redundant []RedundantIndexInfo

	for table, tableIndexes := range grouped {
		colMap := BuildIndexColumnsMap(tableIndexes)

		// 收集所有索引名
		indexNames := make([]string, 0, len(colMap))
		for name := range colMap {
			indexNames = append(indexNames, name)
		}
		sort.Strings(indexNames)

		// 两两比较
		for _, nameA := range indexNames {
			colsA := colMap[nameA]
			for _, nameB := range indexNames {
				if nameA == nameB {
					continue
				}
				colsB := colMap[nameB]

				// 检查 B 是否是 A 的前缀（含完全相同）
				if isPrefix(colsA, colsB) {
					// 确定哪个是冗余的：
					// - 如果 B 是 PRIMARY，B 不算冗余（主键有约束意义）
					// - 如果 A 和 B 列完全相同，优先标记非主键的那个
					if nameB == "PRIMARY" {
						continue
					}
					// 如果列完全相同且 A 也是非主键，只标记其中一个（按名称排序取后者）
					if len(colsA) == len(colsB) && nameA > nameB {
						continue
					}

					redundant = append(redundant, RedundantIndexInfo{
						Table:          table,
						RedundantIndex: nameB,
						CoveredBy:      nameA,
						WastedColumns:  colsB,
					})
				}
			}
		}
	}

	return redundant
}

// isPrefix 检查 target 是否是 source 的前缀
// 即 source[:len(target)] == target
func isPrefix(source, target []string) bool {
	if len(target) > len(source) {
		return false // target 不能比 source 长
	}
	if len(target) == 0 {
		return false
	}
	for i := 0; i < len(target); i++ {
		if source[i] != target[i] {
			return false
		}
	}
	return true
}

// ==================== 纯函数：复合索引建议 ====================

// SuggestCompositeIndexes 根据慢查询模式建议复合索引
// 分析慢查询 WHERE 子句中引用的多个列，如果这些列没有合适的复合索引则建议创建
func SuggestCompositeIndexes(queries []SlowQueryInfo, entries []ShowIndexEntry) []CompositeIndexSuggestion {
	if len(queries) == 0 {
		return nil
	}

	grouped := GroupIndexesByTable(entries)
	var suggestions []CompositeIndexSuggestion
	seen := make(map[string]bool) // 去重键: "table:col1,col2"

	for _, q := range queries {
		// 从 SQL 中提取表名
		table := extractTableName(q.SQLText)
		if table == "" {
			continue
		}

		// 提取 WHERE 条件中的列名
		columns := ExtractWhereColumns(q.SQLText, table)
		if len(columns) < 2 {
			continue // 单列不需要复合索引
		}

		// 检查是否已有合适的复合索引
		tableIndexes, ok := grouped[table]
		if !ok {
			continue
		}
		colMap := BuildIndexColumnsMap(tableIndexes)

		// 检查列组合是否已被某个复合索引覆盖
		if isColumnsCovered(columns, colMap) {
			continue
		}

		// 生成去重键
		key := table + ":" + strings.Join(columns, ",")
		if seen[key] {
			continue
		}
		seen[key] = true

		// 计算置信度
		confidence := 0.7
		if q.RowsExamined > 10000 {
			confidence = 0.9
		} else if q.RowsExamined > 1000 {
			confidence = 0.8
		}

		// 生成建议 SQL
		colList := make([]string, len(columns))
		for i, c := range columns {
			colList[i] = quoteIdentifier(c)
		}
		idxName := "idx_" + strings.Join(columns, "_")
		sugSQL := fmt.Sprintf("ALTER TABLE %s ADD INDEX %s (%s);",
			quoteIdentifier(table), idxName, strings.Join(colList, ", "))

		suggestions = append(suggestions, CompositeIndexSuggestion{
			Table:            table,
			SuggestedColumns: columns,
			Reason:           "WHERE 条件多列查询",
			Confidence:       confidence,
			SQL:              sugSQL,
		})
	}

	// 按置信度降序排序
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	return suggestions
}

// isColumnsCovered 检查列组合是否被已有索引的最左前缀覆盖
func isColumnsCovered(columns []string, colMap map[string][]string) bool {
	for _, idxCols := range colMap {
		if len(idxCols) < len(columns) {
			continue
		}
		match := true
		for i, col := range columns {
			if i >= len(idxCols) || idxCols[i] != col {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// ==================== 纯函数：SQL 列提取 ====================

// ExtractWhereColumns 从 SQL 语句中提取 WHERE 条件引用的列名
func ExtractWhereColumns(sql, table string) []string {
	// 提取 WHERE 子句
	whereMatch := reWhereColumns.FindStringSubmatch(sql)
	if len(whereMatch) < 2 {
		return nil
	}

	whereClause := whereMatch[1]

	// 提取列引用
	matches := reColumnRef.FindAllStringSubmatch(whereClause, -1)
	seen := make(map[string]bool)
	var columns []string

	for _, m := range matches {
		if len(m) >= 2 {
			col := m[1]
			// 跳过 SQL 关键字
			upper := strings.ToUpper(col)
			if isSQLKeyword(upper) {
				continue
			}
			if !seen[col] {
				seen[col] = true
				columns = append(columns, col)
			}
		}
	}

	return columns
}

// extractTableName 从 SQL 中提取 FROM 后的表名
func extractTableName(sql string) string {
	matches := reFromTableName.FindStringSubmatch(sql)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// sqlKeywords SQL 关键字集合（避免将关键字误认为列名）
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "AND": true, "OR": true,
	"NOT": true, "IN": true, "BETWEEN": true, "LIKE": true, "IS": true,
	"NULL": true, "TRUE": true, "FALSE": true, "ORDER": true, "BY": true,
	"GROUP": true, "HAVING": true, "LIMIT": true, "OFFSET": true,
	"JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
	"ON": true, "AS": true, "SET": true, "INTO": true, "VALUES": true,
	"INSERT": true, "UPDATE": true, "DELETE": true, "CREATE": true,
	"ALTER": true, "DROP": true, "TABLE": true, "INDEX": true,
	"EXISTS": true, "CASE": true, "WHEN": true, "THEN": true, "ELSE": true,
	"END": true, "ASC": true, "DESC": true, "DISTINCT": true, "ALL": true,
}

// isSQLKeyword 检查是否是 SQL 关键字
func isSQLKeyword(word string) bool {
	return sqlKeywords[word]
}

// ==================== 纯函数：报告格式化 ====================

// FormatIndexReport 格式化索引分析报告
func FormatIndexReport(redundant []RedundantIndexInfo, composite []CompositeIndexSuggestion) string {
	if len(redundant) == 0 && len(composite) == 0 {
		return "未发现索引问题"
	}

	var b strings.Builder
	b.WriteString("=== 索引分析报告 ===\n\n")

	// 冗余索引部分
	if len(redundant) > 0 {
		fmt.Fprintf(&b, "--- 冗余索引 (%d) ---\n", len(redundant))
		for i, r := range redundant {
			fmt.Fprintf(&b, "  [%d] 表: %s | 冗余索引: %s\n", i+1, r.Table, r.RedundantIndex)
			fmt.Fprintf(&b, "      被 %s 覆盖 | 浪费列: %s\n", r.CoveredBy, strings.Join(r.WastedColumns, ", "))
			fmt.Fprintf(&b, "      建议: DROP INDEX %s ON %s;\n", quoteIdentifier(r.RedundantIndex), quoteIdentifier(r.Table))
		}
		b.WriteString("\n")
	}

	// 复合索引建议部分
	if len(composite) > 0 {
		fmt.Fprintf(&b, "--- 复合索引建议 (%d) ---\n", len(composite))
		for i, c := range composite {
			fmt.Fprintf(&b, "  [%d] 表: %s | 置信度: %.0f%%\n", i+1, c.Table, c.Confidence*100)
			fmt.Fprintf(&b, "      建议列: %s\n", strings.Join(c.SuggestedColumns, ", "))
			fmt.Fprintf(&b, "      原因: %s\n", c.Reason)
			fmt.Fprintf(&b, "      SQL: %s\n", c.SQL)
		}
		b.WriteString("\n")
	}

	return b.String()
}
