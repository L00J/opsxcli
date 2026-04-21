package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// shouldSkipFile
// ---------------------------------------------------------------------------

func TestFileSearch_ShouldSkipFile_HiddenFiles(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"dotfile in cwd", ".env", true},
		{"dotfile with path", "/home/user/project/.gitignore", true},
		{"dotfile multi-dot", ".config.local", true},
		{"dotfile in nested dir", "a/b/c/.hidden", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, shouldSkipFile(tc.path))
		})
	}
}

func TestFileSearch_ShouldSkipFile_BinaryExtensions(t *testing.T) {
	// Exhaustively test every extension listed in the source.
	exts := []string{
		".o", ".a", ".so", ".exe", ".dll",
		".dylib", ".zip", ".tar", ".gz",
		".bz2", ".xz", ".7z", ".rar",
		".png", ".jpg", ".jpeg", ".gif",
		".bmp", ".ico", ".webp", ".svg",
		".mp3", ".mp4", ".avi", ".mkv",
		".mov", ".wmv", ".flac", ".wav",
		".pdf", ".doc", ".docx", ".xls",
		".xlsx", ".ppt", ".pptx", ".class",
		".jar", ".war", ".pyc", ".pyd",
		".woff", ".woff2", ".ttf", ".eot",
		".sqlite", ".db",
	}
	for _, ext := range exts {
		t.Run(ext[1:]+"_lower", func(t *testing.T) {
			assert.True(t, shouldSkipFile("foo"+ext), "should skip "+ext)
		})
		t.Run(ext[1:]+"_upper", func(t *testing.T) {
			assert.True(t, shouldSkipFile("foo"+ext), "should skip uppercase "+ext)
		})
		t.Run(ext[1:]+"_with_path", func(t *testing.T) {
			assert.True(t, shouldSkipFile(filepath.Join("some", "dir", "foo"+ext)))
		})
	}
}

func TestFileSearch_ShouldSkipFile_AllowedFiles(t *testing.T) {
	allowed := []string{
		"main.go",
		"README.md",
		"config.yaml",
		"config.yml",
		"config.json",
		"Makefile",
		"script.sh",
		"style.css",
		"index.html",
		"data.toml",
		"app.py",
		"main.rs",
		"lib.js",
		"mod.ts",
		"Dockerfile",
		"file.txt",
		"/abs/path/to/main.go",
	}
	for _, f := range allowed {
		t.Run(filepath.Base(f), func(t *testing.T) {
			assert.False(t, shouldSkipFile(f), "should NOT skip "+f)
		})
	}
}

// ---------------------------------------------------------------------------
// shouldSkipDir
// ---------------------------------------------------------------------------

func TestFileSearch_ShouldSkipDir_HiddenDirs(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"git dir", ".git", true},
		{"svn dir", ".svn", true},
		{"hg dir", ".hg", true},
		{"idea dir", ".idea", true},
		{"vscode dir", ".vscode", true},
		{"vs dir", ".vs", true},
		{"dot-local in path", "/home/user/.cache", true},
		{"dot-config nested", "project/.config", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, shouldSkipDir(tc.path))
		})
	}
}

func TestFileSearch_ShouldSkipDir_NamedSkipDirs(t *testing.T) {
	skipDirs := []string{
		"node_modules",
		"vendor",
		"__pycache__",
		"dist",
		"build",
		"out",
		"bin",
	}
	for _, d := range skipDirs {
		t.Run(d, func(t *testing.T) {
			assert.True(t, shouldSkipDir(d), "should skip "+d)
		})
		t.Run(d+"_with_path", func(t *testing.T) {
			assert.True(t, shouldSkipDir(filepath.Join("/some", "path", d)))
		})
	}
}

func TestFileSearch_ShouldSkipDir_AllowedDirs(t *testing.T) {
	allowed := []string{
		"src",
		"pkg",
		"cmd",
		"internal",
		"test",
		"tests",
		"scripts",
		".", // current dir — special case, should NOT be skipped
	}
	for _, d := range allowed {
		t.Run(d, func(t *testing.T) {
			assert.False(t, shouldSkipDir(d), "should NOT skip "+d)
		})
	}
}

func TestFileSearch_ShouldSkipDir_DotSelfNotSkipped(t *testing.T) {
	// "." is special — the code explicitly allows it
	assert.False(t, shouldSkipDir("."))
}

// ---------------------------------------------------------------------------
// isBinaryFile
// ---------------------------------------------------------------------------

func TestFileSearch_IsBinaryFile_WithNullBytes(t *testing.T) {
	dir := t.TempDir()

	t.Run("contains null byte", func(t *testing.T) {
		path := filepath.Join(dir, "binary.dat")
		// 512 bytes with a null byte in the middle
		data := make([]byte, 512)
		data[100] = 0x00
		require.NoError(t, os.WriteFile(path, data, 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.True(t, isBinaryFile(f))
	})

	t.Run("null byte at start", func(t *testing.T) {
		path := filepath.Join(dir, "nullstart.bin")
		data := []byte{0x00, 0x01, 0x02, 0x03}
		require.NoError(t, os.WriteFile(path, data, 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.True(t, isBinaryFile(f))
	})

	t.Run("null byte at end", func(t *testing.T) {
		path := filepath.Join(dir, "nullend.bin")
		data := make([]byte, 512)
		data[511] = 0x00
		require.NoError(t, os.WriteFile(path, data, 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.True(t, isBinaryFile(f))
	})
}

func TestFileSearch_IsBinaryFile_PlainText(t *testing.T) {
	dir := t.TempDir()

	t.Run("plain text content", func(t *testing.T) {
		path := filepath.Join(dir, "hello.txt")
		require.NoError(t, os.WriteFile(path, []byte("Hello, World! This is plain text."), 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.False(t, isBinaryFile(f))
	})

	t.Run("go source code", func(t *testing.T) {
		path := filepath.Join(dir, "main.go")
		content := []byte("package main\n\nfunc main() {\n    println(\"hi\")\n}\n")
		require.NoError(t, os.WriteFile(path, content, 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.False(t, isBinaryFile(f))
	})

	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(dir, "empty.txt")
		require.NoError(t, os.WriteFile(path, []byte{}, 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.False(t, isBinaryFile(f))
	})

	t.Run("UTF-8 with high bytes but no null", func(t *testing.T) {
		path := filepath.Join(dir, "utf8.txt")
		// Chinese characters — no null bytes
		content := []byte("这是一个测试文件，包含中文字符。")
		require.NoError(t, os.WriteFile(path, content, 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.False(t, isBinaryFile(f))
	})
}

func TestFileSearch_IsBinaryFile_ShortFile(t *testing.T) {
	dir := t.TempDir()

	t.Run("short binary file (< 512 bytes) with null", func(t *testing.T) {
		path := filepath.Join(dir, "short.bin")
		require.NoError(t, os.WriteFile(path, []byte{0x01, 0x02, 0x00, 0x04}, 0o644))

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		assert.True(t, isBinaryFile(f))
	})
}
