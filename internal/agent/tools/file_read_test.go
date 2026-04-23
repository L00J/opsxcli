package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// detectBinaryFile
// ---------------------------------------------------------------------------

func TestFileRead_detectBinaryFile_TextFile(t *testing.T) {
	// A plain-text file should NOT be detected as binary
	f, err := os.CreateTemp(t.TempDir(), "text-*.txt")
	require.NoError(t, err)
	defer f.Close()

	_, err = f.WriteString("Hello, world!\nThis is plain text.\n")
	require.NoError(t, err)

	// Rewind so detectBinaryFile can read from the start
	_, err = f.Seek(0, 0)
	require.NoError(t, err)

	isBin, err := detectBinaryFile(f)
	assert.NoError(t, err)
	assert.False(t, isBin, "plain text file should not be detected as binary")
}

func TestFileRead_detectBinaryFile_BinaryFile(t *testing.T) {
	// A file containing at least one null byte should be detected as binary
	f, err := os.CreateTemp(t.TempDir(), "bin-*.bin")
	require.NoError(t, err)
	defer f.Close()

	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0xFF, 0xD8} // PNG-like header with nulls
	_, err = f.Write(data)
	require.NoError(t, err)

	_, err = f.Seek(0, 0)
	require.NoError(t, err)

	isBin, err := detectBinaryFile(f)
	assert.NoError(t, err)
	assert.True(t, isBin, "file with null bytes should be detected as binary")
}

func TestFileRead_detectBinaryFile_EmptyFile(t *testing.T) {
	// An empty file has no null bytes => not binary
	f, err := os.CreateTemp(t.TempDir(), "empty-*.txt")
	require.NoError(t, err)
	defer f.Close()

	// File is empty, no write needed but we need to ensure offset=0
	_, err = f.Seek(0, 0)
	require.NoError(t, err)

	isBin, err := detectBinaryFile(f)
	assert.NoError(t, err)
	assert.False(t, isBin, "empty file should not be detected as binary")
}

func TestFileRead_detectBinaryFile_AllNullBytes(t *testing.T) {
	// A file of all null bytes must be binary
	f, err := os.CreateTemp(t.TempDir(), "null-*.bin")
	require.NoError(t, err)
	defer f.Close()

	nulls := make([]byte, 512) // all zero
	_, err = f.Write(nulls)
	require.NoError(t, err)

	_, err = f.Seek(0, 0)
	require.NoError(t, err)

	isBin, err := detectBinaryFile(f)
	assert.NoError(t, err)
	assert.True(t, isBin)
}

func TestFileRead_detectBinaryFile_NullByteAtEnd(t *testing.T) {
	// Null byte at position 511 (last of the 512-byte check window)
	f, err := os.CreateTemp(t.TempDir(), "trailing-null-*.bin")
	require.NoError(t, err)
	defer f.Close()

	data := make([]byte, 512)
	for i := range data {
		data[i] = 'A'
	}
	data[511] = 0x00 // null at the very end

	_, err = f.Write(data)
	require.NoError(t, err)

	_, err = f.Seek(0, 0)
	require.NoError(t, err)

	isBin, err := detectBinaryFile(f)
	assert.NoError(t, err)
	assert.True(t, isBin, "null byte within first 512 bytes should be detected")
}

// ---------------------------------------------------------------------------
// readFileLines
// ---------------------------------------------------------------------------

func TestFileRead_readFileLines_Basic(t *testing.T) {
	input := "line1\nline2\nline3\n"
	reader := strings.NewReader(input)

	lines, total, truncated, err := readFileLines(reader, 1, 100)
	require.NoError(t, err)

	assert.Equal(t, 3, total)
	assert.False(t, truncated)
	require.Len(t, lines, 3)

	assert.Equal(t, 1, lines[0].Num)
	assert.Equal(t, "line1", lines[0].Content)
	assert.Equal(t, 2, lines[1].Num)
	assert.Equal(t, "line2", lines[1].Content)
	assert.Equal(t, 3, lines[2].Num)
	assert.Equal(t, "line3", lines[2].Content)
}

func TestFileRead_readFileLines_WithOffset(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5\n"
	reader := strings.NewReader(input)

	lines, total, truncated, err := readFileLines(reader, 3, 100)
	require.NoError(t, err)

	assert.Equal(t, 5, total)
	assert.False(t, truncated)
	require.Len(t, lines, 3)

	assert.Equal(t, 3, lines[0].Num)
	assert.Equal(t, "line3", lines[0].Content)
	assert.Equal(t, 5, lines[2].Num)
	assert.Equal(t, "line5", lines[2].Content)
}

func TestFileRead_readFileLines_WithLimit(t *testing.T) {
	input := "line1\nline2\nline3\nline4\nline5\n"
	reader := strings.NewReader(input)

	lines, total, truncated, err := readFileLines(reader, 1, 2)
	require.NoError(t, err)

	assert.Equal(t, 5, total) // total counts ALL lines
	assert.True(t, truncated)
	require.Len(t, lines, 2)

	assert.Equal(t, 1, lines[0].Num)
	assert.Equal(t, "line1", lines[0].Content)
	assert.Equal(t, 2, lines[1].Num)
	assert.Equal(t, "line2", lines[1].Content)
}

func TestFileRead_readFileLines_OffsetAndLimit(t *testing.T) {
	input := "a\nb\nc\nd\ne\nf\ng\n"
	reader := strings.NewReader(input)

	lines, total, truncated, err := readFileLines(reader, 3, 2)
	require.NoError(t, err)

	assert.Equal(t, 7, total)
	assert.True(t, truncated)
	require.Len(t, lines, 2)

	assert.Equal(t, 3, lines[0].Num)
	assert.Equal(t, "c", lines[0].Content)
	assert.Equal(t, 4, lines[1].Num)
	assert.Equal(t, "d", lines[1].Content)
}

func TestFileRead_readFileLines_EmptyInput(t *testing.T) {
	reader := strings.NewReader("")

	lines, total, truncated, err := readFileLines(reader, 1, 100)
	require.NoError(t, err)

	assert.Equal(t, 0, total)
	assert.False(t, truncated)
	assert.Empty(t, lines)
}

func TestFileRead_readFileLines_OffsetBeyondContent(t *testing.T) {
	input := "only\none\n"
	reader := strings.NewReader(input)

	lines, total, truncated, err := readFileLines(reader, 100, 10)
	require.NoError(t, err)

	assert.Equal(t, 2, total)
	assert.False(t, truncated)
	assert.Empty(t, lines, "offset beyond file should return no lines")
}

func TestFileRead_readFileLines_NoTrailingNewline(t *testing.T) {
	input := "first line\nsecond line" // no trailing \n
	reader := strings.NewReader(input)

	lines, total, truncated, err := readFileLines(reader, 1, 100)
	require.NoError(t, err)

	assert.Equal(t, 2, total)
	assert.False(t, truncated)
	require.Len(t, lines, 2)
	assert.Equal(t, "first line", lines[0].Content)
	assert.Equal(t, "second line", lines[1].Content)
}

func TestFileRead_readFileLines_LimitEqualsTotal(t *testing.T) {
	// When limit exactly equals the total number of lines, no truncation
	input := "a\nb\nc\n"
	reader := strings.NewReader(input)

	lines, total, truncated, err := readFileLines(reader, 1, 3)
	require.NoError(t, err)

	assert.Equal(t, 3, total)
	assert.False(t, truncated)
	require.Len(t, lines, 3)
}

// ---------------------------------------------------------------------------
// expandPath
// ---------------------------------------------------------------------------

func TestFileRead_expandPath_Tilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := expandPath("~/Documents")
	expected := filepath.Join(home, "Documents")
	assert.Equal(t, expected, result)
}

func TestFileRead_expandPath_JustTilde(t *testing.T) {
	// "~" alone (without trailing slash) is NOT expanded — only "~/..." is
	// But expandPath also calls filepath.Abs, so it becomes an absolute path.
	result := expandPath("~")
	// Should at minimum be an absolute path (filepath.Abs resolves it)
	assert.True(t, filepath.IsAbs(result), "result should be an absolute path: %s", result)
}

func TestFileRead_expandPath_EnvVar(t *testing.T) {
	os.Setenv("TEST_OPSX_READ_DIR", "/tmp/opsxtest")
	defer os.Unsetenv("TEST_OPSX_READ_DIR")

	result := expandPath("$TEST_OPSX_READ_DIR/subdir")
	assert.Equal(t, "/tmp/opsxtest/subdir", result)
}

func TestFileRead_expandPath_AbsolutePath(t *testing.T) {
	result := expandPath("/usr/local/bin")
	assert.Equal(t, "/usr/local/bin", result)
}

func TestFileRead_expandPath_RelativePath(t *testing.T) {
	// Relative paths get resolved to absolute by filepath.Abs
	result := expandPath("relative/path")
	assert.True(t, filepath.IsAbs(result), "relative path should be resolved to absolute: %s", result)
	assert.Contains(t, result, "relative/path")
}

func TestFileRead_expandPath_TildeWithEnvVar(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	os.Setenv("TEST_OPSX_SUBDIR", "mysubdir")
	defer os.Unsetenv("TEST_OPSX_SUBDIR")

	result := expandPath("~/$TEST_OPSX_SUBDIR")
	expected := filepath.Join(home, "mysubdir")
	assert.Equal(t, expected, result)
}

func TestFileRead_expandPath_EmptyString(t *testing.T) {
	result := expandPath("")
	// Should return the absolute path of cwd (filepath.Abs("") gives cwd)
	assert.True(t, filepath.IsAbs(result), "empty string should resolve to absolute cwd path")
}

// ---------------------------------------------------------------------------
// formatFileSize
// ---------------------------------------------------------------------------

func TestFileRead_formatFileSize_Bytes(t *testing.T) {
	assert.Equal(t, "0 B", formatFileSize(0))
	assert.Equal(t, "1 B", formatFileSize(1))
	assert.Equal(t, "512 B", formatFileSize(512))
	assert.Equal(t, "1023 B", formatFileSize(1023))
}

func TestFileRead_formatFileSize_KB(t *testing.T) {
	assert.Equal(t, "1.0 KB", formatFileSize(1024))
	assert.Equal(t, "1.5 KB", formatFileSize(1536))         // 1.5 * 1024
	assert.Equal(t, "512.0 KB", formatFileSize(512*1024))   // 512 KB
	assert.Equal(t, "1023.0 KB", formatFileSize(1023*1024)) // just under 1 MB
}

func TestFileRead_formatFileSize_MB(t *testing.T) {
	assert.Equal(t, "1.0 MB", formatFileSize(1024*1024))
	assert.Equal(t, "1.5 MB", formatFileSize(int64(1.5*1024*1024)))
	assert.Equal(t, "512.0 MB", formatFileSize(512*1024*1024))
}

func TestFileRead_formatFileSize_GB(t *testing.T) {
	assert.Equal(t, "1.0 GB", formatFileSize(1024*1024*1024))
	assert.Equal(t, "2.5 GB", formatFileSize(int64(2.5*1024*1024*1024)))
	assert.Equal(t, "1024.0 GB", formatFileSize(1024*1024*1024*1024)) // 1 TB expressed in GB
}

func TestFileRead_formatFileSize_BoundaryValues(t *testing.T) {
	// Exact boundary at 1 KB
	assert.Equal(t, "1.0 KB", formatFileSize(1024))
	// Exact boundary at 1 MB
	assert.Equal(t, "1.0 MB", formatFileSize(1024*1024))
	// Exact boundary at 1 GB
	assert.Equal(t, "1.0 GB", formatFileSize(1024*1024*1024))
	// One byte below 1 KB
	assert.Equal(t, "1023 B", formatFileSize(1023))
}
