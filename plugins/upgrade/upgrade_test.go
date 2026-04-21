package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- convertOSName ---

func TestConvertOSName_Linux(t *testing.T) {
	assert.Equal(t, "Linux", convertOSName("linux"))
}

func TestConvertOSName_Darwin(t *testing.T) {
	assert.Equal(t, "Darwin", convertOSName("darwin"))
}

func TestConvertOSName_Windows(t *testing.T) {
	assert.Equal(t, "Windows", convertOSName("windows"))
}

func TestConvertOSName_Empty(t *testing.T) {
	assert.Equal(t, "", convertOSName(""))
}

// --- convertArch ---

func TestConvertArch_Amd64(t *testing.T) {
	assert.Equal(t, "x86_64", convertArch("linux", "amd64"))
	assert.Equal(t, "x86_64", convertArch("darwin", "amd64"))
	assert.Equal(t, "x86_64", convertArch("windows", "amd64"))
}

func TestConvertArch_Arm64_Linux(t *testing.T) {
	assert.Equal(t, "aarch64", convertArch("linux", "arm64"))
}

func TestConvertArch_Arm64_Darwin(t *testing.T) {
	assert.Equal(t, "arm64", convertArch("darwin", "arm64"))
}

func TestConvertArch_386(t *testing.T) {
	assert.Equal(t, "i386", convertArch("linux", "386"))
}

func TestConvertArch_Unknown(t *testing.T) {
	assert.Equal(t, "mips", convertArch("linux", "mips"))
}

// --- detectInstallPath ---

func TestDetectInstallPath_StandardPath(t *testing.T) {
	// When current path is already standard
	assert.Equal(t, "/usr/local/bin/opsxcli", detectInstallPath("/usr/local/bin/opsxcli"))
	assert.Equal(t, "/usr/bin/opsxcli", detectInstallPath("/usr/bin/opsxcli"))
}

func TestDetectInstallPath_NonStandardPath(t *testing.T) {
	// When current path is non-standard, it should try to find standard paths
	// or fall back to current path
	result := detectInstallPath("/home/user/opsxcli")
	// Should return either a standard path or the original
	assert.NotEmpty(t, result)
}
