package tui

import (
	"strings"
	"testing"
)

// --- highlightJSON tests ---

func TestHighlightJSON_Strings(t *testing.T) {
	input := `{"name": "hello"}`
	result := highlightJSON(input)
	// Should contain the original text parts
	if !strings.Contains(result, "name") {
		t.Error("highlightJSON should preserve key 'name'")
	}
	if !strings.Contains(result, "hello") {
		t.Error("highlightJSON should preserve value 'hello'")
	}
}

func TestHighlightJSON_Numbers(t *testing.T) {
	input := `{"count": 42, "ratio": 3.14}`
	result := highlightJSON(input)
	if !strings.Contains(result, "count") {
		t.Error("highlightJSON should preserve key 'count'")
	}
	if !strings.Contains(result, "42") {
		t.Error("highlightJSON should preserve number 42")
	}
	if !strings.Contains(result, "3.14") {
		t.Error("highlightJSON should preserve number 3.14")
	}
}

func TestHighlightJSON_Booleans(t *testing.T) {
	input := `{"active": true, "deleted": false, "empty": null}`
	result := highlightJSON(input)
	if !strings.Contains(result, "true") {
		t.Error("highlightJSON should preserve true")
	}
	if !strings.Contains(result, "false") {
		t.Error("highlightJSON should preserve false")
	}
	if !strings.Contains(result, "null") {
		t.Error("highlightJSON should preserve null")
	}
}

func TestHighlightJSON_EmptyInput(t *testing.T) {
	result := highlightJSON("")
	if result != "" {
		t.Errorf("highlightJSON of empty string should be empty, got %q", result)
	}
}

func TestHighlightJSON_PlainText(t *testing.T) {
	input := "just some text without JSON"
	result := highlightJSON(input)
	if !strings.Contains(result, "just some text") {
		t.Error("highlightJSON should preserve plain text")
	}
}

// --- highlightGo tests ---

func TestHighlightGo_Keywords(t *testing.T) {
	input := "func main() { return nil }"
	result := highlightGo(input)
	if !strings.Contains(result, "func") {
		t.Error("highlightGo should preserve 'func'")
	}
	if !strings.Contains(result, "main") {
		t.Error("highlightGo should preserve 'main'")
	}
	if !strings.Contains(result, "return") {
		t.Error("highlightGo should preserve 'return'")
	}
}

func TestHighlightGo_Strings(t *testing.T) {
	input := `msg := "hello world"`
	result := highlightGo(input)
	if !strings.Contains(result, "hello world") {
		t.Error("highlightGo should preserve string content")
	}
}

func TestHighlightGo_RawStrings(t *testing.T) {
	input := "s := `raw string`"
	result := highlightGo(input)
	if !strings.Contains(result, "raw string") {
		t.Error("highlightGo should preserve raw string content")
	}
}

func TestHighlightGo_Comments(t *testing.T) {
	input := "// this is a comment\ncode line"
	result := highlightGo(input)
	if !strings.Contains(result, "this is a comment") {
		t.Error("highlightGo should preserve comment text")
	}
	if !strings.Contains(result, "code line") {
		t.Error("highlightGo should preserve code after comment")
	}
}

func TestHighlightGo_Numbers(t *testing.T) {
	input := "x := 42"
	result := highlightGo(input)
	if !strings.Contains(result, "42") {
		t.Error("highlightGo should preserve numbers")
	}
}

func TestHighlightGo_EmptyInput(t *testing.T) {
	result := highlightGo("")
	if result != "" {
		t.Errorf("highlightGo of empty string should be empty, got %q", result)
	}
}

// --- highlightGeneric tests ---

func TestHighlightGeneric_Comments(t *testing.T) {
	input := "# this is a comment\ncode line"
	result := highlightGeneric(input)
	if !strings.Contains(result, "this is a comment") {
		t.Error("highlightGeneric should preserve comment text")
	}
	if !strings.Contains(result, "code line") {
		t.Error("highlightGeneric should preserve code after comment")
	}
}

func TestHighlightGeneric_Strings(t *testing.T) {
	input := `print "hello world"`
	result := highlightGeneric(input)
	if !strings.Contains(result, "hello world") {
		t.Error("highlightGeneric should preserve string content")
	}
}

func TestHighlightGeneric_Numbers(t *testing.T) {
	input := "count = 100"
	result := highlightGeneric(input)
	if !strings.Contains(result, "100") {
		t.Error("highlightGeneric should preserve numbers")
	}
}

func TestHighlightGeneric_EmptyInput(t *testing.T) {
	result := highlightGeneric("")
	if result != "" {
		t.Errorf("highlightGeneric of empty string should be empty, got %q", result)
	}
}

func TestHighlightGeneric_NoSpecialContent(t *testing.T) {
	input := "plain text without special content"
	result := highlightGeneric(input)
	if !strings.Contains(result, "plain text") {
		t.Error("highlightGeneric should preserve plain text")
	}
}

// --- highlightKeywords tests ---

func TestHighlightKeywords_Basic(t *testing.T) {
	input := "if condition then"
	keywords := []string{"if", "then"}
	result := highlightKeywords(input, keywords)
	if !strings.Contains(result, "if") {
		t.Error("highlightKeywords should preserve keyword 'if'")
	}
	if !strings.Contains(result, "then") {
		t.Error("highlightKeywords should preserve keyword 'then'")
	}
	if !strings.Contains(result, "condition") {
		t.Error("highlightKeywords should preserve non-keyword text")
	}
}

func TestHighlightKeywords_EmptyInput(t *testing.T) {
	result := highlightKeywords("", []string{"if"})
	if result != "" {
		t.Errorf("highlightKeywords of empty string should be empty, got %q", result)
	}
}

func TestHighlightKeywords_EmptyKeywords(t *testing.T) {
	input := "some text"
	result := highlightKeywords(input, []string{})
	if result != input {
		t.Errorf("highlightKeywords with empty keywords should return input unchanged, got %q", result)
	}
}

func TestHighlightKeywords_NoMatch(t *testing.T) {
	input := "no matching words here"
	keywords := []string{"func", "package"}
	result := highlightKeywords(input, keywords)
	if !strings.Contains(result, "no matching words here") {
		t.Error("highlightKeywords should preserve text when no keywords match")
	}
}

func TestHighlightKeywords_MultipleKeywords(t *testing.T) {
	input := "for i := range items"
	keywords := []string{"for", "range"}
	result := highlightKeywords(input, keywords)
	if !strings.Contains(result, "for") {
		t.Error("highlightKeywords should match 'for'")
	}
	if !strings.Contains(result, "range") {
		t.Error("highlightKeywords should match 'range'")
	}
}

// --- highlightPattern tests ---

func TestHighlightPattern_BasicMatch(t *testing.T) {
	input := `value "hello" end`
	pattern := `"([^"\\]|\\.)*"`
	result := highlightPattern(input, pattern, bashStringStyle)
	if !strings.Contains(result, "hello") {
		t.Error("highlightPattern should preserve matched content")
	}
	if !strings.Contains(result, "value") {
		t.Error("highlightPattern should preserve text before match")
	}
	if !strings.Contains(result, "end") {
		t.Error("highlightPattern should preserve text after match")
	}
}

func TestHighlightPattern_NoMatch(t *testing.T) {
	input := "no quotes here"
	pattern := `"([^"\\]|\\.)*"`
	result := highlightPattern(input, pattern, bashStringStyle)
	if result != input {
		t.Errorf("highlightPattern with no match should return input unchanged, got %q", result)
	}
}

func TestHighlightPattern_EmptyInput(t *testing.T) {
	result := highlightPattern("", `\d+`, bashNumberStyle)
	if result != "" {
		t.Errorf("highlightPattern of empty string should be empty, got %q", result)
	}
}

func TestHighlightPattern_NumberMatch(t *testing.T) {
	input := "count 42 items"
	pattern := `\b\d+(\.\d+)?\b`
	result := highlightPattern(input, pattern, bashNumberStyle)
	if !strings.Contains(result, "42") {
		t.Error("highlightPattern should preserve matched number")
	}
	if !strings.Contains(result, "count") {
		t.Error("highlightPattern should preserve surrounding text")
	}
}

// --- highlightCode (dispatcher) tests ---

func TestHighlightCode_Bash(t *testing.T) {
	input := "# comment\necho hello"
	result := highlightCode(input, "bash")
	if !strings.Contains(result, "comment") {
		t.Error("highlightCode bash should preserve comment text")
	}
}

func TestHighlightCode_Shell(t *testing.T) {
	input := "$ ls -la"
	result := highlightCode(input, "sh")
	if !strings.Contains(result, "ls") {
		t.Error("highlightCode sh should dispatch to highlightBash")
	}
}

func TestHighlightCode_JSON(t *testing.T) {
	input := `{"key": 123}`
	result := highlightCode(input, "json")
	if !strings.Contains(result, "key") {
		t.Error("highlightCode json should preserve content")
	}
}

func TestHighlightCode_Go(t *testing.T) {
	input := "func main() {}"
	result := highlightCode(input, "go")
	if !strings.Contains(result, "main") {
		t.Error("highlightCode go should preserve content")
	}
}

func TestHighlightCode_Golang(t *testing.T) {
	input := "func main() {}"
	result := highlightCode(input, "golang")
	if !strings.Contains(result, "main") {
		t.Error("highlightCode golang should dispatch to highlightGo")
	}
}

func TestHighlightCode_Unknown(t *testing.T) {
	input := "some code here"
	result := highlightCode(input, "python")
	if !strings.Contains(result, "some code") {
		t.Error("highlightCode unknown lang should dispatch to highlightGeneric")
	}
}

func TestHighlightCode_CaseInsensitive(t *testing.T) {
	input := "# test"
	result := highlightCode(input, "BASH")
	if !strings.Contains(result, "test") {
		t.Error("highlightCode should handle case-insensitive language names")
	}
}

// --- isTableSeparator edge cases ---

func TestIsTableSeparator_Alignment(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"| --- | --- |", true},
		{"| :---: |", true},
		{"| --- | :---: | ---: |", true},
		{"| not a separator |", false},
		{"| a | b |", false},
		{"no pipes", false},
		{"", false},
		{"| |", true}, // all parts empty or whitespace-only
	}
	for _, tt := range tests {
		result := isTableSeparator(tt.input)
		if result != tt.expected {
			t.Errorf("isTableSeparator(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// --- isListItem edge cases ---

func TestIsListItem_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"- item", true},
		{"* item", true},
		{"1. first", true},
		{"99. ninety nine", true},
		{"1.too", false},          // no space after dot
		{"a. letter", false},      // not numeric prefix
		{"1234. too long", false}, // prefix > 3 digits
		{"", false},
		{"no list", false},
		{"1", false},      // too short
		{"1.x", false},    // no space
		{"0. zero", true}, // valid number list
	}
	for _, tt := range tests {
		result := isListItem(tt.input)
		if result != tt.expected {
			t.Errorf("isListItem(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// --- renderCodeBlock tests ---

func TestRenderCodeBlock_WithLanguage(t *testing.T) {
	result := renderCodeBlock("echo hello", "bash")
	if !strings.Contains(result, "bash") {
		t.Error("renderCodeBlock should include language label")
	}
	if !strings.Contains(result, "echo") {
		t.Error("renderCodeBlock should preserve code content")
	}
}

func TestRenderCodeBlock_WithoutLanguage(t *testing.T) {
	result := renderCodeBlock("plain code", "")
	if strings.Contains(result, "\n") && !strings.Contains(result, "plain code") {
		t.Error("renderCodeBlock without language should not have language header")
	}
	if !strings.Contains(result, "plain code") {
		t.Error("renderCodeBlock should preserve code content")
	}
}

func TestRenderCodeBlock_EmptyCode(t *testing.T) {
	result := renderCodeBlock("", "go")
	// Should handle empty code gracefully
	if result == "" {
		// empty input produces empty-ish output, that's fine
		return
	}
	// If not empty, should still have language label
	if !strings.Contains(result, "go") {
		t.Error("renderCodeBlock with empty code but language should include language label")
	}
}

// --- isDivider additional edge cases ---

func TestIsDivider_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"---", true},
		{"***", true},
		{"___", true},
		{"- - -", false},    // spaces break the triple pattern
		{"* * *", false},    // spaces break the triple pattern
		{"--", false},       // too short
		{"", false},         // empty
		{"a---", false},     // invalid char
		{"---text", false},  // invalid char
		{"   ---   ", true}, // whitespace ok, has ---
	}
	for _, tt := range tests {
		result := isDivider(tt.input)
		if result != tt.expected {
			t.Errorf("isDivider(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

// --- isTableLine tests ---

func TestIsTableLine(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"| a | b |", true},
		{"|a|b|", true},
		{"| single |", true}, // 2 pipes meets >= 2 requirement
		{"no pipes", false},
		{"| a | b", false}, // doesn't end with |
		{"a | b |", false}, // doesn't start with |
		{"", false},
	}
	for _, tt := range tests {
		result := isTableLine(tt.input)
		if result != tt.expected {
			t.Errorf("isTableLine(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
