package builtin

import (
	"bytes"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== free 测试 ====================

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestFree_Basic(t *testing.T) {
	output := captureStdout(func() {
		err := Free(FreeOptions{})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Mem:")
	assert.Contains(t, output, "Swap:")
}

func TestFree_HumanReadable(t *testing.T) {
	output := captureStdout(func() {
		err := Free(FreeOptions{Human: true})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Mem:")
}

func TestFree_WithTotal(t *testing.T) {
	output := captureStdout(func() {
		err := Free(FreeOptions{Total: true})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Total:")
}

func TestFree_UnitBytes(t *testing.T) {
	output := captureStdout(func() {
		err := Free(FreeOptions{Unit: "b"})
		assert.NoError(t, err)
	})
	assert.NotEmpty(t, output)
}

func TestFree_UnitMB(t *testing.T) {
	output := captureStdout(func() {
		err := Free(FreeOptions{Unit: "m"})
		assert.NoError(t, err)
	})
	assert.NotEmpty(t, output)
}

func TestFree_UnitGB(t *testing.T) {
	output := captureStdout(func() {
		err := Free(FreeOptions{Unit: "g"})
		assert.NoError(t, err)
	})
	assert.NotEmpty(t, output)
}

// ==================== df 测试 ====================

func TestDf_Basic(t *testing.T) {
	output := captureStdout(func() {
		err := Df(DfOptions{})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Filesystem")
}

func TestDf_HumanReadable(t *testing.T) {
	output := captureStdout(func() {
		err := Df(DfOptions{Human: true})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Filesystem")
}

func TestDf_ShowFsType(t *testing.T) {
	output := captureStdout(func() {
		err := Df(DfOptions{FsType: true})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "Type")
}

// ==================== kill 测试 ====================

func TestKill_ListSignals(t *testing.T) {
	output := captureStdout(func() {
		err := Kill([]string{}, KillOptions{List: true})
		assert.NoError(t, err)
	})
	assert.Contains(t, output, "TERM")
	assert.Contains(t, output, "KILL")
	assert.Contains(t, output, "HUP")
}

func TestKill_NoPID(t *testing.T) {
	err := Kill([]string{}, KillOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "需要指定 PID")
}

func TestKill_InvalidPID(t *testing.T) {
	err := Kill([]string{"not_a_number"}, KillOptions{})
	// Kill 对无效 PID 打印 stderr 错误，并返回空字符串 error
	assert.Error(t, err)
}

// ==================== parseSignal 测试 ====================

func TestParseSignal_ByName(t *testing.T) {
	sig, err := parseSignal("TERM")
	assert.NoError(t, err)
	assert.Equal(t, syscall.SIGTERM, sig)

	sig, err = parseSignal("KILL")
	assert.NoError(t, err)
	assert.Equal(t, syscall.SIGKILL, sig)

	sig, err = parseSignal("HUP")
	assert.NoError(t, err)
	assert.Equal(t, syscall.SIGHUP, sig)
}

func TestParseSignal_WithSigPrefix(t *testing.T) {
	sig, err := parseSignal("SIGTERM")
	assert.NoError(t, err)
	assert.Equal(t, syscall.SIGTERM, sig)
}

func TestParseSignal_CaseInsensitive(t *testing.T) {
	sig, err := parseSignal("term")
	assert.NoError(t, err)
	assert.Equal(t, syscall.SIGTERM, sig)

	sig, err = parseSignal("sigkill")
	assert.NoError(t, err)
	assert.Equal(t, syscall.SIGKILL, sig)
}

func TestParseSignal_ByNumber(t *testing.T) {
	sig, err := parseSignal("15") // SIGTERM
	assert.NoError(t, err)
	assert.Equal(t, syscall.SIGTERM, sig)
}

func TestParseSignal_Unknown(t *testing.T) {
	_, err := parseSignal("UNKNOWN_SIGNAL")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未知信号")
}

// ==================== isVirtualFS 测试 ====================

func TestIsVirtualFS(t *testing.T) {
	assert.True(t, isVirtualFS("sysfs"))
	assert.True(t, isVirtualFS("proc"))
	assert.True(t, isVirtualFS("tmpfs"))
	assert.True(t, isVirtualFS("devtmpfs"))
	assert.True(t, isVirtualFS("cgroup"))
	assert.True(t, isVirtualFS("overlay"))
	assert.False(t, isVirtualFS("ext4"))
	assert.False(t, isVirtualFS("xfs"))
	assert.False(t, isVirtualFS("apfs"))
	assert.False(t, isVirtualFS("ntfs"))
}

// ==================== formatMemoryValue 测试 ====================

func TestFormatMemoryValue_Default(t *testing.T) {
	fn := formatMemoryValue(0, FreeOptions{})
	// 默认 KB
	assert.Equal(t, "1024", fn(1024*1024))
}

func TestFormatMemoryValue_Human(t *testing.T) {
	fn := formatMemoryValue(0, FreeOptions{Human: true})
	result := fn(1024 * 1024)
	assert.Contains(t, result, "Mi")
}

func TestFormatMemoryValue_Bytes(t *testing.T) {
	fn := formatMemoryValue(0, FreeOptions{Unit: "b"})
	assert.Equal(t, "1048576", fn(1024*1024))
}

func TestFormatMemoryValue_KB(t *testing.T) {
	fn := formatMemoryValue(0, FreeOptions{Unit: "k"})
	assert.Equal(t, "1024", fn(1024*1024))
}

func TestFormatMemoryValue_MB(t *testing.T) {
	fn := formatMemoryValue(0, FreeOptions{Unit: "m"})
	assert.Equal(t, "1", fn(1024*1024))
}

func TestFormatMemoryValue_GB(t *testing.T) {
	fn := formatMemoryValue(0, FreeOptions{Unit: "g"})
	assert.Equal(t, "1", fn(1024*1024*1024))
}
