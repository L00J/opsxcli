package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetFormat(t *testing.T) {
	SetFormat(FormatJSON)
	if !IsJSON() {
		t.Error("IsJSON should return true after SetFormat(FormatJSON)")
	}
	SetFormat(FormatText)
	if IsJSON() {
		t.Error("IsJSON should return false after SetFormat(FormatText)")
	}
}

func TestSetQuiet(t *testing.T) {
	SetQuiet(true)
	if !IsQuiet() {
		t.Error("IsQuiet should return true after SetQuiet(true)")
	}
	SetQuiet(false)
	if IsQuiet() {
		t.Error("IsQuiet should return false after SetQuiet(false)")
	}
}

func TestPrint_TextMode(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)
	SetFormat(FormatText)
	SetQuiet(false)

	Print("hello world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("expected 'hello world' in output, got: %s", buf.String())
	}
}

func TestPrint_QuietMode(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)
	SetFormat(FormatText)
	SetQuiet(true)

	Print("should not appear")
	if buf.Len() != 0 {
		t.Errorf("expected nothing in quiet mode, got: %s", buf.String())
	}
}

func TestPrint_JSONMode(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)
	SetFormat(FormatJSON)
	SetQuiet(false)

	Print(map[string]string{"key": "value"})
	got := buf.String()
	if !strings.Contains(got, `"key"`) {
		t.Errorf("expected JSON output, got: %s", got)
	}
}

func TestJSON_Output(t *testing.T) {
	buf := &bytes.Buffer{}
	SetWriter(buf)

	JSON(struct{ Name string }{"test"})
	got := buf.String()
	if !strings.Contains(got, `"Name"`) || !strings.Contains(got, `"test"`) {
		t.Errorf("expected JSON with Name field, got: %s", got)
	}
}

func TestResult_Struct(t *testing.T) {
	r := Result{
		Success:  true,
		Message:  "done",
		Data:     map[string]int{"count": 42},
		ExitCode: 0,
	}
	if !r.Success {
		t.Error("Result.Success should be true")
	}
	if r.ExitCode != 0 {
		t.Errorf("Result.ExitCode = %d, want 0", r.ExitCode)
	}
}
