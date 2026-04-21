package core

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBuildInfo(t *testing.T) {
	bi := NewBuildInfo("1.0.0", "2026-01-01", "abc1234")
	assert.NotNil(t, bi)
	assert.Equal(t, "1.0.0", bi.Version)
	assert.Equal(t, "2026-01-01", bi.BuildTime)
	assert.Equal(t, "abc1234", bi.GitCommit)
}

func TestBuildInfo_String_WithCommit(t *testing.T) {
	bi := &BuildInfo{Version: "0.4.0", GitCommit: "deadbeef"}
	s := bi.String()
	assert.Contains(t, s, "0.4.0")
	assert.Contains(t, s, "deadbeef")
	assert.Contains(t, s, "commit:")
}

func TestBuildInfo_String_NoCommit(t *testing.T) {
	bi := &BuildInfo{Version: "0.4.0", GitCommit: ""}
	assert.Equal(t, "0.4.0", bi.String())
}

func TestBuildInfo_String_UnknownCommit(t *testing.T) {
	bi := &BuildInfo{Version: "0.4.0", GitCommit: "unknown"}
	assert.Equal(t, "0.4.0", bi.String())
}

func TestBuildInfo_All(t *testing.T) {
	bi := NewBuildInfo("2.0.0", "2026-04-21", "a1b2c3d")
	m := bi.All()
	assert.Equal(t, "2.0.0", m["version"])
	assert.Equal(t, "2026-04-21", m["buildTime"])
	assert.Equal(t, "a1b2c3d", m["gitCommit"])
	assert.Equal(t, runtime.Version(), m["goVersion"])
	expectedPlatform := runtime.GOOS + "/" + runtime.GOARCH
	assert.Equal(t, expectedPlatform, m["platform"])
}

func TestBuildInfo_All_FieldsCount(t *testing.T) {
	bi := NewBuildInfo("1.0", "t", "g")
	m := bi.All()
	assert.Len(t, m, 5)
}

