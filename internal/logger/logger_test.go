package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	Init()
	if infoLogger == nil {
		t.Error("infoLogger should not be nil after Init")
	}
	Close()
}

func TestSetDebug(t *testing.T) {
	Init()
	SetDebug(true)
	if !debugMode {
		t.Error("debugMode should be true after SetDebug(true)")
	}
	SetDebug(false)
	if debugMode {
		t.Error("debugMode should be false after SetDebug(false)")
	}
	Close()
}

func TestSetQuiet(t *testing.T) {
	InitWithOptions(false, false)
	SetQuiet(true)
	if !quietMode {
		t.Error("quietMode should be true after SetQuiet(true)")
	}
	SetQuiet(false)
	if quietMode {
		t.Error("quietMode should be false after SetQuiet(false)")
	}
	Close()
}

func TestSetNoColor(t *testing.T) {
	InitWithOptions(false, false)
	SetNoColor(true)
	if !noColor {
		t.Error("noColor should be true after SetNoColor(true)")
	}
	SetNoColor(false)
	if noColor {
		t.Error("noColor should be false after SetNoColor(false)")
	}
	Close()
}

func TestIsQuiet_IsNoColor(t *testing.T) {
	InitWithOptions(true, true)
	if !IsQuiet() {
		t.Error("IsQuiet should return true when quietMode is set")
	}
	if !IsNoColor() {
		t.Error("IsNoColor should return true when noColor is set")
	}
	Close()
}

func TestLogFileCreated(t *testing.T) {
	Init()
	logDir := filepath.Join(os.Getenv("HOME"), ".opsxcli", "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("log directory should be created")
	}
	Close()
}

func TestClose_Idempotent(t *testing.T) {
	Init()
	Close()
	Close()
}
