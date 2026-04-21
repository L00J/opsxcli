package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== Busybox Archive 命令测试 =====

func TestNewTarCmd_Basic(t *testing.T) {
	cmd := NewTarCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewGzipCmd_Basic(t *testing.T) {
	cmd := NewGzipCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewUnzipCmd_Basic(t *testing.T) {
	cmd := NewUnzipCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Busybox Disk 命令测试 =====

func TestNewDdCmd_Basic(t *testing.T) {
	cmd := NewDdCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Busybox File 命令测试 =====

func TestNewLsCmd_Basic(t *testing.T) {
	cmd := NewLsCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewCpCmd_Basic(t *testing.T) {
	cmd := NewCpCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewMvCmd_Basic(t *testing.T) {
	cmd := NewMvCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewRmCmd_Basic(t *testing.T) {
	cmd := NewRmCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewMkdirCmd_Basic(t *testing.T) {
	cmd := NewMkdirCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewRmdirCmd_Basic(t *testing.T) {
	cmd := NewRmdirCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewTreeCmd_Basic(t *testing.T) {
	cmd := NewTreeCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewTouchCmd_Basic(t *testing.T) {
	cmd := NewTouchCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewChmodCmd_Basic(t *testing.T) {
	cmd := NewChmodCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewChownCmd_Basic(t *testing.T) {
	cmd := NewChownCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewLnCmd_Basic(t *testing.T) {
	cmd := NewLnCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewCatCmd_Basic(t *testing.T) {
	cmd := NewCatCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Busybox Network 命令测试 =====

func TestNewIfconfigCmd_Basic(t *testing.T) {
	cmd := NewIfconfigCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewRouteCmd_Basic(t *testing.T) {
	cmd := NewRouteCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewIpCmd_Basic(t *testing.T) {
	cmd := NewIpCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Busybox Process 命令测试 =====

func TestNewPsCmd_Basic(t *testing.T) {
	cmd := NewPsCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewTopCmd_Basic(t *testing.T) {
	cmd := NewTopCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewKillCmd_Basic(t *testing.T) {
	cmd := NewKillCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewPstreeCmd_Basic(t *testing.T) {
	cmd := NewPstreeCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Busybox System 命令测试 =====

func TestNewUnameCmd_Basic(t *testing.T) {
	cmd := NewUnameCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewHostnameCmd_Basic(t *testing.T) {
	cmd := NewHostnameCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewWhoamiCmd_Basic(t *testing.T) {
	cmd := NewWhoamiCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewIdCmd_Basic(t *testing.T) {
	cmd := NewIdCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewFreeCmd_Basic(t *testing.T) {
	cmd := NewFreeCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewDfCmd_Basic(t *testing.T) {
	cmd := NewDfCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewDuCmd_Basic(t *testing.T) {
	cmd := NewDuCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Busybox Text 命令测试 =====

func TestNewHeadCmd_Basic(t *testing.T) {
	cmd := NewHeadCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewTailCmd_Basic(t *testing.T) {
	cmd := NewTailCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewGrepCmd_Basic(t *testing.T) {
	cmd := NewGrepCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Busybox Time 命令测试 =====

func TestNewDateCmd_Basic(t *testing.T) {
	cmd := NewDateCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

func TestNewSleepCmd_Basic(t *testing.T) {
	cmd := NewSleepCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}
