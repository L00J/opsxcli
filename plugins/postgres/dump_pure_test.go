package postgres

import (
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// filterTables
// ---------------------------------------------------------------------------

func TestFilterTables_AllPass(t *testing.T) {
	tables := []string{"users", "orders", "products"}
	opts := DefaultDumpOptions()
	result := filterTables(tables, opts)
	if len(result) != 3 {
		t.Fatalf("expected 3 tables, got %d: %v", len(result), result)
	}
}

func TestFilterTables_IgnoreOne(t *testing.T) {
	tables := []string{"users", "orders", "products"}
	opts := DumpOptions{
		Format:     "sql",
		WithData:   true,
		WithSchema: true,
		IgnoreTables: []string{"orders"},
	}
	result := filterTables(tables, opts)
	if len(result) != 2 {
		t.Fatalf("expected 2 tables, got %d: %v", len(result), result)
	}
	for _, v := range result {
		if v == "orders" {
			t.Fatal("orders should have been filtered out")
		}
	}
}

func TestFilterTables_IgnoreAll(t *testing.T) {
	tables := []string{"users", "orders"}
	opts := DumpOptions{
		IgnoreTables: []string{"users", "orders"},
	}
	result := filterTables(tables, opts)
	if len(result) != 0 {
		t.Fatalf("expected 0 tables, got %d: %v", len(result), result)
	}
}

func TestFilterTables_SpecifyInclude(t *testing.T) {
	tables := []string{"users", "orders", "products"}
	opts := DumpOptions{
		Tables: []string{"users", "products"},
	}
	result := filterTables(tables, opts)
	if len(result) != 2 {
		t.Fatalf("expected 2 tables, got %d: %v", len(result), result)
	}
	for _, v := range result {
		if v == "orders" {
			t.Fatal("orders should not be included")
		}
	}
}

func TestFilterTables_IncludeAndIgnore(t *testing.T) {
	tables := []string{"users", "orders", "products"}
	opts := DumpOptions{
		Tables:       []string{"users", "orders"},
		IgnoreTables: []string{"orders"},
	}
	result := filterTables(tables, opts)
	if len(result) != 1 || result[0] != "users" {
		t.Fatalf("expected only [users], got %v", result)
	}
}

func TestFilterTables_EmptyInput(t *testing.T) {
	opts := DefaultDumpOptions()
	result := filterTables(nil, opts)
	if len(result) != 0 {
		t.Fatalf("expected 0 tables, got %d", len(result))
	}
}

func TestFilterTables_IncludeNonExistent(t *testing.T) {
	tables := []string{"users"}
	opts := DumpOptions{
		Tables: []string{"nonexistent"},
	}
	result := filterTables(tables, opts)
	if len(result) != 0 {
		t.Fatalf("expected 0 tables, got %d: %v", len(result), result)
	}
}

// ---------------------------------------------------------------------------
// pgFormatSQLValue
// ---------------------------------------------------------------------------

func TestPgFormatSQLValue_Nil(t *testing.T) {
	if got := pgFormatSQLValue(nil); got != "NULL" {
		t.Fatalf("expected NULL, got %s", got)
	}
}

func TestPgFormatSQLValue_String(t *testing.T) {
	if got := pgFormatSQLValue("hello"); got != "'hello'" {
		t.Fatalf("expected 'hello', got %s", got)
	}
}

func TestPgFormatSQLValue_Int64(t *testing.T) {
	if got := pgFormatSQLValue(int64(42)); got != "42" {
		t.Fatalf("expected 42, got %s", got)
	}
}

func TestPgFormatSQLValue_Float64(t *testing.T) {
	got := pgFormatSQLValue(float64(3.14))
	if got != "3.14" {
		t.Fatalf("expected 3.14, got %s", got)
	}
}

func TestPgFormatSQLValue_Bool(t *testing.T) {
	if got := pgFormatSQLValue(true); got != "true" {
		t.Fatalf("expected true, got %s", got)
	}
	if got := pgFormatSQLValue(false); got != "false" {
		t.Fatalf("expected false, got %s", got)
	}
}

func TestPgFormatSQLValue_Time(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if got := pgFormatSQLValue(ts); got != "'2024-01-15 10:30:00'" {
		t.Fatalf("expected '2024-01-15 10:30:00', got %s", got)
	}
}

func TestPgFormatSQLValue_ByteSlice(t *testing.T) {
	if got := pgFormatSQLValue([]byte("abc")); got != "'abc'" {
		t.Fatalf("expected 'abc', got %s", got)
	}
}

func TestPgFormatSQLValue_StringWithQuote(t *testing.T) {
	if got := pgFormatSQLValue("it's"); got != "'it''s'" {
		t.Fatalf("expected 'it''s', got %s", got)
	}
}

func TestPgFormatSQLValue_DefaultType(t *testing.T) {
	// uint 用 default 分支
	if got := pgFormatSQLValue(uint(7)); got != "'7'" {
		t.Fatalf("expected '7', got %s", got)
	}
}

// ---------------------------------------------------------------------------
// pgFormatCSVValue
// ---------------------------------------------------------------------------

func TestPgFormatCSVValue_Nil(t *testing.T) {
	if got := pgFormatCSVValue(nil); got != "" {
		t.Fatalf("expected empty string, got %s", got)
	}
}

func TestPgFormatCSVValue_String(t *testing.T) {
	if got := pgFormatCSVValue("hello"); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

func TestPgFormatCSVValue_Int64(t *testing.T) {
	if got := pgFormatCSVValue(int64(42)); got != "42" {
		t.Fatalf("expected 42, got %s", got)
	}
}

func TestPgFormatCSVValue_Float64(t *testing.T) {
	if got := pgFormatCSVValue(float64(3.14)); got != "3.14" {
		t.Fatalf("expected 3.14, got %s", got)
	}
}

func TestPgFormatCSVValue_Bool(t *testing.T) {
	if got := pgFormatCSVValue(true); got != "t" {
		t.Fatalf("expected t, got %s", got)
	}
	if got := pgFormatCSVValue(false); got != "f" {
		t.Fatalf("expected f, got %s", got)
	}
}

func TestPgFormatCSVValue_Time(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if got := pgFormatCSVValue(ts); got != "2024-01-15 10:30:00" {
		t.Fatalf("expected 2024-01-15 10:30:00, got %s", got)
	}
}

func TestPgFormatCSVValue_ByteSlice(t *testing.T) {
	if got := pgFormatCSVValue([]byte("abc")); got != "abc" {
		t.Fatalf("expected abc, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// pgEscapeString
// ---------------------------------------------------------------------------

func TestPgEscapeString_SingleQuote(t *testing.T) {
	if got := pgEscapeString("it's"); got != "it''s" {
		t.Fatalf("expected it''s, got %s", got)
	}
}

func TestPgEscapeString_Backslash(t *testing.T) {
	if got := pgEscapeString(`a\b`); got != `a\\b` {
		t.Fatalf("expected a\\\\b, got %s", got)
	}
}

func TestPgEscapeString_Newline(t *testing.T) {
	if got := pgEscapeString("a\nb"); got != `a\nb` {
		t.Fatalf("expected a\\nb, got %q", got)
	}
}

func TestPgEscapeString_Tab(t *testing.T) {
	if got := pgEscapeString("a\tb"); got != `a\tb` {
		t.Fatalf("expected a\\tb, got %q", got)
	}
}

func TestPgEscapeString_CarriageReturn(t *testing.T) {
	if got := pgEscapeString("a\rb"); got != `a\rb` {
		t.Fatalf("expected a\\rb, got %q", got)
	}
}

func TestPgEscapeString_NoEscapeNeeded(t *testing.T) {
	if got := pgEscapeString("hello"); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// pgQuoteIdentifier
// ---------------------------------------------------------------------------

func TestPgQuoteIdentifier_Normal(t *testing.T) {
	if got := pgQuoteIdentifier("users"); got != `"users"` {
		t.Fatalf("expected \"users\", got %s", got)
	}
}

func TestPgQuoteIdentifier_ContainsDoubleQuote(t *testing.T) {
	if got := pgQuoteIdentifier(`my"table`); got != `"my""table"` {
		t.Fatalf("expected \"my\"\"table\", got %s", got)
	}
}

// ---------------------------------------------------------------------------
// splitSQL
// ---------------------------------------------------------------------------

func TestSplitSQL_Simple(t *testing.T) {
	stmts := splitSQL("SELECT 1; SELECT 2;")
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
	if stmts[0] != "SELECT 1" {
		t.Fatalf("unexpected first statement: %s", stmts[0])
	}
	if stmts[1] != "SELECT 2" {
		t.Fatalf("unexpected second statement: %s", stmts[1])
	}
}

func TestSplitSQL_SemicolonInSingleQuote(t *testing.T) {
	stmts := splitSQL(`INSERT INTO t VALUES ('a;b');`)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_SemicolonInEscapedQuotes(t *testing.T) {
	// PostgreSQL 用 '' 转义单引号
	stmts := splitSQL(`INSERT INTO t VALUES ('it''s;ok');`)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_DollarQuoting(t *testing.T) {
	input := `CREATE FUNCTION foo() RETURNS void AS $$ BEGIN SELECT 1; END; $$ LANGUAGE plpgsql;`
	stmts := splitSQL(input)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_TaggedDollarQuoting(t *testing.T) {
	input := `CREATE FUNCTION foo() RETURNS void AS $body$ BEGIN SELECT 1; END; $body$ LANGUAGE plpgsql;`
	stmts := splitSQL(input)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_LineComment(t *testing.T) {
	stmts := splitSQL("SELECT 1; -- this is a comment\nSELECT 2;")
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_BlockComment(t *testing.T) {
	stmts := splitSQL("SELECT 1; /* comment; here */ SELECT 2;")
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_EmptyInput(t *testing.T) {
	stmts := splitSQL("")
	if len(stmts) != 0 {
		t.Fatalf("expected 0 statements, got %d", len(stmts))
	}
}

func TestSplitSQL_TrailingSemicolon(t *testing.T) {
	stmts := splitSQL("SELECT 1;;; SELECT 2;")
	// 连续分号之间会产生空语句，splitSQL 会 TrimSpace 后跳过空串
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_NoTrailingSemicolon(t *testing.T) {
	stmts := splitSQL("SELECT 1")
	if len(stmts) != 1 || stmts[0] != "SELECT 1" {
		t.Fatalf("expected [SELECT 1], got %v", stmts)
	}
}

func TestSplitSQL_CommentWithSemicolon(t *testing.T) {
	stmts := splitSQL("-- a;b\nSELECT 1;")
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
}

// ---------------------------------------------------------------------------
// truncateString
// ---------------------------------------------------------------------------

func TestTruncateString_Short(t *testing.T) {
	if got := truncateString("hello", 10); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

func TestTruncateString_Exact(t *testing.T) {
	if got := truncateString("hello", 5); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

func TestTruncateString_Long(t *testing.T) {
	longStr := strings.Repeat("a", 200)
	got := truncateString(longStr, 100)
	if len(got) != 103 { // 100 + "..."
		t.Fatalf("expected length 103, got %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatal("expected ... suffix")
	}
}

func TestTruncateString_Zero(t *testing.T) {
	got := truncateString("hello", 0)
	if got != "..." {
		t.Fatalf("expected ..., got %s", got)
	}
}

// ---------------------------------------------------------------------------
// DefaultDumpOptions
// ---------------------------------------------------------------------------

func TestDumpDefaultDumpOptions(t *testing.T) {
	opts := DefaultDumpOptions()
	if opts.Format != "sql" {
		t.Fatalf("expected Format=sql, got %s", opts.Format)
	}
	if !opts.WithData {
		t.Fatal("expected WithData=true")
	}
	if !opts.WithSchema {
		t.Fatal("expected WithSchema=true")
	}
	if len(opts.Tables) != 0 {
		t.Fatalf("expected empty Tables, got %v", opts.Tables)
	}
	if len(opts.IgnoreTables) != 0 {
		t.Fatalf("expected empty IgnoreTables, got %v", opts.IgnoreTables)
	}
}
