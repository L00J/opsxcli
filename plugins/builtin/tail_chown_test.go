package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- tailReader (fileops.go) ---

func TestTailReader_Basic(t *testing.T) {
	r := strings.NewReader("line1\nline2\nline3\n")
	err := tailReader(r, "test", 10, false)
	assert.NoError(t, err)
}

func TestTailReader_LessThanN(t *testing.T) {
	r := strings.NewReader("line1\nline2\n")
	err := tailReader(r, "test", 10, false)
	assert.NoError(t, err)
}

func TestTailReader_MoreThanN(t *testing.T) {
	var buf bytes.Buffer
	for i := 0; i < 20; i++ {
		buf.WriteString("line\n")
	}
	err := tailReader(&buf, "test", 5, false)
	assert.NoError(t, err)
}

func TestTailReader_Empty(t *testing.T) {
	r := strings.NewReader("")
	err := tailReader(r, "test", 10, false)
	assert.NoError(t, err)
}

func TestTailReader_SingleLine(t *testing.T) {
	r := strings.NewReader("only line\n")
	err := tailReader(r, "test", 10, false)
	assert.NoError(t, err)
}

func TestTailReader_ExactN(t *testing.T) {
	var buf bytes.Buffer
	for i := 0; i < 5; i++ {
		buf.WriteString("line\n")
	}
	err := tailReader(&buf, "test", 5, false)
	assert.NoError(t, err)
}

// --- Tail (fileops.go) ---

func TestTail_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(f, []byte("a\nb\nc\n"), 0644))
	err := Tail([]string{f}, 2)
	assert.NoError(t, err)
}

func TestTail_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	require.NoError(t, os.WriteFile(f1, []byte("x\ny\n"), 0644))
	require.NoError(t, os.WriteFile(f2, []byte("p\nq\n"), 0644))
	err := Tail([]string{f1, f2}, 1)
	assert.NoError(t, err)
}

func TestTail_NonexistentFile(t *testing.T) {
	err := Tail([]string{"/nonexistent/file.txt"}, 10)
	assert.NoError(t, err)
}

func TestTail_DefaultLines(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	var buf bytes.Buffer
	for i := 0; i < 20; i++ {
		buf.WriteString("line\n")
	}
	require.NoError(t, os.WriteFile(f, buf.Bytes(), 0644))
	err := Tail([]string{f}, 0)
	assert.NoError(t, err)
}

// --- Chown / chownRecursive (fileops.go) ---

func TestChown_NonRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0644))
	_ = Chown("0", []string{f}, false)
}

func TestChown_Recursive(t *testing.T) {
	tmpDir := t.TempDir()
	sub := filepath.Join(tmpDir, "sub")
	require.NoError(t, os.Mkdir(sub, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "file.txt"), []byte("data"), 0644))
	_ = Chown("0:0", []string{tmpDir}, true)
}

func TestChown_InvalidPath(t *testing.T) {
	err := Chown("0", []string{"/nonexistent/path"}, false)
	assert.NoError(t, err)
}

func TestChown_OwnerGroup(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0644))
	_ = Chown("1000:1000", []string{f}, false)
}

// --- headReader (fileops.go) ---

func TestHeadReader_Basic(t *testing.T) {
	r := strings.NewReader("line1\nline2\nline3\n")
	err := headReader(r, "test", 2, false)
	assert.NoError(t, err)
}

func TestHeadReader_MoreThanAvailable(t *testing.T) {
	r := strings.NewReader("line1\n")
	err := headReader(r, "test", 10, false)
	assert.NoError(t, err)
}

func TestHeadReader_Empty(t *testing.T) {
	r := strings.NewReader("")
	err := headReader(r, "test", 10, false)
	assert.NoError(t, err)
}
