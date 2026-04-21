package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ===== Builtin Disk 命令测试 =====

func TestNewDdCmd_Basic(t *testing.T) {
	cmd := NewDdCmd()
	assert.NotNil(t, cmd)
	assert.NotEmpty(t, cmd.Short)
}

// ===== Builtin File 命令测试 =====

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

// ===== Builtin Network 命令测试 =====

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

// ===== Builtin Text 命令测试 =====

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
