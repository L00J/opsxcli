package builtin

import (
	"testing"
)

// ---------------------------------------------------------------------------
// network.go — 依赖系统调用，不 panic 即可
// ---------------------------------------------------------------------------

func TestShowInterfaces_Coverage(t *testing.T) {
	_ = ShowInterfaces()
}

func TestShowRoutes_Coverage(t *testing.T) {
	_ = ShowRoutes()
}

func TestShowIPAddr_Coverage(t *testing.T) {
	_ = ShowIPAddr()
}

// ---------------------------------------------------------------------------
// process.go — 依赖系统调用
// ---------------------------------------------------------------------------

func TestPs_Basic(t *testing.T) {
	_ = Ps(PsOptions{})
}

func TestPs_Full(t *testing.T) {
	_ = Ps(PsOptions{Full: true})
}

func TestPs_AllUsers(t *testing.T) {
	_ = Ps(PsOptions{AllUsers: true})
}

func TestPstree_Basic(t *testing.T) {
	_ = Pstree(PstreeOptions{})
}

func TestPstree_ShowPID(t *testing.T) {
	_ = Pstree(PstreeOptions{ShowPID: true})
}

func TestTop_Basic(t *testing.T) {
	_ = Top(TopOptions{})
}

func TestTop_Count1(t *testing.T) {
	_ = Top(TopOptions{Count: 1})
}
