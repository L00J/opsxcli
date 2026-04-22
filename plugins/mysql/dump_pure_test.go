package mysql

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
// formatSQLValue
// ---------------------------------------------------------------------------

func TestFormatSQLValue_Nil(t *testing.T) {
	if got := formatSQLValue(nil); got != "NULL" {
		t.Fatalf("expected NULL, got %s", got)
	}
}

func TestFormatSQLValue_String(t *testing.T) {
	if got := formatSQLValue("hello"); got != "'hello'" {
		t.Fatalf("expected 'hello', got %s", got)
	}
}

func TestFormatSQLValue_Int64(t *testing.T) {
	if got := formatSQLValue(int64(42)); got != "42" {
		t.Fatalf("expected 42, got %s", got)
	}
}

func TestFormatSQLValue_Float64(t *testing.T) {
	if got := formatSQLValue(float64(3.14)); got != "3.14" {
		t.Fatalf("expected 3.14, got %s", got)
	}
}

func TestFormatSQLValue_Bool(t *testing.T) {
	if got := formatSQLValue(true); got != "1" {
		t.Fatalf("expected 1, got %s", got)
	}
	if got := formatSQLValue(false); got != "0" {
		t.Fatalf("expected 0, got %s", got)
	}
}

func TestFormatSQLValue_Time(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if got := formatSQLValue(ts); got != "'2024-01-15 10:30:00'" {
		t.Fatalf("expected '2024-01-15 10:30:00', got %s", got)
	}
}

func TestFormatSQLValue_ByteSlice(t *testing.T) {
	if got := formatSQLValue([]byte("abc")); got != "'abc'" {
		t.Fatalf("expected 'abc', got %s", got)
	}
}

func TestFormatSQLValue_StringWithQuote(t *testing.T) {
	if got := formatSQLValue("it's"); got != `'it\'s'` {
		t.Fatalf("expected 'it\\'s', got %s", got)
	}
}

func TestFormatSQLValue_DefaultType(t *testing.T) {
	if got := formatSQLValue(uint(7)); got != "'7'" {
		t.Fatalf("expected '7', got %s", got)
	}
}

// ---------------------------------------------------------------------------
// formatCSVValue
// ---------------------------------------------------------------------------

func TestFormatCSVValue_Nil(t *testing.T) {
	if got := formatCSVValue(nil); got != "" {
		t.Fatalf("expected empty string, got %s", got)
	}
}

func TestFormatCSVValue_String(t *testing.T) {
	if got := formatCSVValue("hello"); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

func TestFormatCSVValue_Int64(t *testing.T) {
	if got := formatCSVValue(int64(42)); got != "42" {
		t.Fatalf("expected 42, got %s", got)
	}
}

func TestFormatCSVValue_Float64(t *testing.T) {
	if got := formatCSVValue(float64(3.14)); got != "3.14" {
		t.Fatalf("expected 3.14, got %s", got)
	}
}

func TestFormatCSVValue_Bool(t *testing.T) {
	if got := formatCSVValue(true); got != "1" {
		t.Fatalf("expected 1, got %s", got)
	}
	if got := formatCSVValue(false); got != "0" {
		t.Fatalf("expected 0, got %s", got)
	}
}

func TestFormatCSVValue_Time(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	if got := formatCSVValue(ts); got != "2024-01-15 10:30:00" {
		t.Fatalf("expected 2024-01-15 10:30:00, got %s", got)
	}
}

func TestFormatCSVValue_ByteSlice(t *testing.T) {
	if got := formatCSVValue([]byte("abc")); got != "abc" {
		t.Fatalf("expected abc, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// escapeString
// ---------------------------------------------------------------------------

func TestEscapeString_Backslash(t *testing.T) {
	if got := escapeString(`a\b`); got != `a\\b` {
		t.Fatalf("expected a\\\\b, got %s", got)
	}
}

func TestEscapeString_SingleQuote(t *testing.T) {
	if got := escapeString("it's"); got != `it\'s` {
		t.Fatalf("expected it\\'s, got %s", got)
	}
}

func TestEscapeString_Newline(t *testing.T) {
	if got := escapeString("a\nb"); got != `a\nb` {
		t.Fatalf("expected a\\nb, got %q", got)
	}
}

func TestEscapeString_CarriageReturn(t *testing.T) {
	if got := escapeString("a\rb"); got != `a\rb` {
		t.Fatalf("expected a\\rb, got %q", got)
	}
}

func TestEscapeString_Tab(t *testing.T) {
	if got := escapeString("a\tb"); got != `a\tb` {
		t.Fatalf("expected a\\tb, got %q", got)
	}
}

func TestEscapeString_NullByte(t *testing.T) {
	if got := escapeString("a\x00b"); got != `a\0b` {
		t.Fatalf("expected a\\0b, got %q", got)
	}
}

func TestEscapeString_NoEscapeNeeded(t *testing.T) {
	if got := escapeString("hello"); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

// ---------------------------------------------------------------------------
// quoteIdentifier
// ---------------------------------------------------------------------------

func TestQuoteIdentifier_Normal(t *testing.T) {
	if got := quoteIdentifier("users"); got != "`users`" {
		t.Fatalf("expected `users`, got %s", got)
	}
}

func TestQuoteIdentifier_ContainsBacktick(t *testing.T) {
	if got := quoteIdentifier("my`table"); got != "`my``table`" {
		t.Fatalf("expected `my``table`, got %s", got)
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

func TestSplitSQL_SemicolonInDoubleQuote(t *testing.T) {
	stmts := splitSQL(`INSERT INTO t VALUES ("a;b");`)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
}

func TestSplitSQL_EscapedSingleQuote(t *testing.T) {
	stmts := splitSQL(`INSERT INTO t VALUES ('it\'s;ok');`)
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

func TestSplitSQL_TrailingSemicolons(t *testing.T) {
	stmts := splitSQL("SELECT 1;;; SELECT 2;")
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
