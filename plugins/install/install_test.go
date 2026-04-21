package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- OSInfo struct ---

func TestOSInfo_Fields(t *testing.T) {
	info := &OSInfo{
		ID:         "ubuntu",
		VersionID:  "20.04",
		Name:       "Ubuntu",
		PrettyName: "Ubuntu 20.04 LTS",
	}
	assert.Equal(t, "ubuntu", info.ID)
	assert.Equal(t, "20.04", info.VersionID)
	assert.Equal(t, "Ubuntu", info.Name)
	assert.Equal(t, "Ubuntu 20.04 LTS", info.PrettyName)
}

func TestOSInfo_Empty(t *testing.T) {
	info := &OSInfo{}
	assert.Empty(t, info.ID)
	assert.Empty(t, info.VersionID)
	assert.Empty(t, info.Name)
	assert.Empty(t, info.PrettyName)
}

// --- parseOSReleaseLines ---

func TestParseOSReleaseLines_Ubuntu(t *testing.T) {
	content := `ID=ubuntu
VERSION_ID="22.04"
NAME="Ubuntu"
PRETTY_NAME="Ubuntu 22.04.3 LTS"
`
	info, err := parseOSReleaseLines(content, "/etc/os-release")
	require.NoError(t, err)
	assert.Equal(t, "ubuntu", info.ID)
	assert.Equal(t, "22.04", info.VersionID)
	assert.Equal(t, "Ubuntu", info.Name)
	assert.Equal(t, "Ubuntu 22.04.3 LTS", info.PrettyName)
}

func TestParseOSReleaseLines_CentOS(t *testing.T) {
	content := `ID="centos"
VERSION_ID="7"
NAME="CentOS Linux"
PRETTY_NAME="CentOS Linux 7 (Core)"
`
	info, err := parseOSReleaseLines(content, "/etc/os-release")
	require.NoError(t, err)
	assert.Equal(t, "centos", info.ID)
	assert.Equal(t, "7", info.VersionID)
	assert.Equal(t, "CentOS Linux", info.Name)
	assert.Equal(t, "CentOS Linux 7 (Core)", info.PrettyName)
}

func TestParseOSReleaseLines_Debian(t *testing.T) {
	content := `12.5
`
	info, err := parseOSReleaseLines(content, "/etc/debian_version")
	require.NoError(t, err)
	assert.Equal(t, "debian", info.ID)
	assert.Equal(t, "12.5", info.VersionID)
	assert.Equal(t, "Debian 12.5", info.PrettyName)
}

func TestParseOSReleaseLines_Debian_Multiline(t *testing.T) {
	content := `11.6
extra line
`
	info, err := parseOSReleaseLines(content, "/etc/debian_version")
	require.NoError(t, err)
	assert.Equal(t, "debian", info.ID)
	assert.Equal(t, "11.6", info.VersionID)
}

func TestParseOSReleaseLines_Debian_Empty(t *testing.T) {
	info, err := parseOSReleaseLines("", "/etc/debian_version")
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestParseOSReleaseLines_RedHat_CentOS7(t *testing.T) {
	content := `CentOS Linux release 7.9.2009 (Core)
`
	info, err := parseOSReleaseLines(content, "/etc/redhat-release")
	require.NoError(t, err)
	assert.Equal(t, "centos", info.ID)
	assert.Equal(t, "7", info.VersionID)
	assert.Contains(t, info.PrettyName, "CentOS")
}

func TestParseOSReleaseLines_RedHat_CentOS6(t *testing.T) {
	content := `CentOS release 6.10 (Final)
`
	info, err := parseOSReleaseLines(content, "/etc/redhat-release")
	require.NoError(t, err)
	assert.Equal(t, "centos", info.ID)
	assert.Equal(t, "6", info.VersionID)
}

func TestParseOSReleaseLines_RedHat_RHEL(t *testing.T) {
	content := `Red Hat Enterprise Linux release 8.5 (Ootpa)
`
	info, err := parseOSReleaseLines(content, "/etc/redhat-release")
	require.NoError(t, err)
	assert.Equal(t, "rhel", info.ID)
}

func TestParseOSReleaseLines_RedHat_Unknown(t *testing.T) {
	content := `Some Unknown Linux
`
	info, err := parseOSReleaseLines(content, "/etc/redhat-release")
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestParseOSReleaseLines_RedHat_Empty(t *testing.T) {
	info, err := parseOSReleaseLines("", "/etc/redhat-release")
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestParseOSReleaseLines_OSRelease_Empty(t *testing.T) {
	info, err := parseOSReleaseLines("", "/etc/os-release")
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestParseOSReleaseLines_OSRelease_Comments(t *testing.T) {
	content := `# This is a comment
ID=rocky
# Another comment
VERSION_ID="9.2"
NAME="Rocky Linux"
PRETTY_NAME="Rocky Linux 9.2 (Blue Onyx)"
`
	info, err := parseOSReleaseLines(content, "/usr/lib/os-release")
	require.NoError(t, err)
	assert.Equal(t, "rocky", info.ID)
	assert.Equal(t, "9.2", info.VersionID)
	assert.Equal(t, "Rocky Linux", info.Name)
}

func TestParseOSReleaseLines_OSRelease_NoQuotes(t *testing.T) {
	content := `ID=debian
VERSION_ID=12
NAME=Debian GNU/Linux
PRETTY_NAME=Debian GNU/Linux 12 (bookworm)
`
	info, err := parseOSReleaseLines(content, "/etc/os-release")
	require.NoError(t, err)
	assert.Equal(t, "debian", info.ID)
	assert.Equal(t, "12", info.VersionID)
}

func TestParseOSReleaseLines_OSRelease_EmptyLines(t *testing.T) {
	content := `ID=fedora

VERSION_ID="39"

PRETTY_NAME=Fedora Linux 39

`
	info, err := parseOSReleaseLines(content, "/etc/os-release")
	require.NoError(t, err)
	assert.Equal(t, "fedora", info.ID)
	assert.Equal(t, "39", info.VersionID)
}

func TestParseOSReleaseLines_OSRelease_NoID(t *testing.T) {
	content := `VERSION_ID="1.0"
NAME=TestOS
`
	info, err := parseOSReleaseLines(content, "/etc/os-release")
	assert.Error(t, err)
	assert.Nil(t, info)
}

func TestParseOSReleaseLines_OSRelease_InvalidLine(t *testing.T) {
	content := `ID=alpine
no_equals_sign
VERSION_ID="3.19"
`
	info, err := parseOSReleaseLines(content, "/etc/os-release")
	require.NoError(t, err)
	assert.Equal(t, "alpine", info.ID)
	assert.Equal(t, "3.19", info.VersionID)
}

// --- GetPackageManager ---

func TestGetPackageManager_CentOS(t *testing.T) {
	info := &OSInfo{ID: "centos", PrettyName: "CentOS Linux 7 (Core)", Name: "CentOS Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "yum", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_RHEL(t *testing.T) {
	info := &OSInfo{ID: "rhel", PrettyName: "Red Hat Enterprise Linux 8", Name: "Red Hat Enterprise Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "yum", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_Rocky(t *testing.T) {
	info := &OSInfo{ID: "rocky", PrettyName: "Rocky Linux 8", Name: "Rocky Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "yum", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_AlmaLinux(t *testing.T) {
	info := &OSInfo{ID: "almalinux", PrettyName: "AlmaLinux 9", Name: "AlmaLinux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "yum", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_Fedora(t *testing.T) {
	info := &OSInfo{ID: "fedora", PrettyName: "Fedora Linux 39", Name: "Fedora Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "dnf", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_Ubuntu(t *testing.T) {
	info := &OSInfo{ID: "ubuntu", PrettyName: "Ubuntu 22.04 LTS", Name: "Ubuntu"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "apt-get", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_Debian(t *testing.T) {
	info := &OSInfo{ID: "debian", PrettyName: "Debian GNU/Linux 12", Name: "Debian GNU/Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "apt-get", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_AmazonLinux(t *testing.T) {
	info := &OSInfo{ID: "amzn", PrettyName: "Amazon Linux 2", Name: "Amazon Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "yum", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_AliyunLinux3_DNF(t *testing.T) {
	info := &OSInfo{ID: "alios", VersionID: "3", PrettyName: "Aliyun Linux 3", Name: "Aliyun Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "dnf", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_AliyunLinux4_DNF(t *testing.T) {
	info := &OSInfo{ID: "aliyun", VersionID: "4", PrettyName: "Aliyun Linux 4", Name: "Aliyun Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "dnf", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_AliyunLinux2_YUM(t *testing.T) {
	info := &OSInfo{ID: "alios", VersionID: "2", PrettyName: "Aliyun Linux 2", Name: "Aliyun Linux"}
	cmd, args, err := GetPackageManager(info)
	require.NoError(t, err)
	assert.Equal(t, "yum", cmd)
	assert.Equal(t, []string{"install"}, args)
}

func TestGetPackageManager_Unsupported(t *testing.T) {
	info := &OSInfo{ID: "freebsd", PrettyName: "FreeBSD 14", Name: "FreeBSD"}
	cmd, args, err := GetPackageManager(info)
	assert.Error(t, err)
	assert.Empty(t, cmd)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), "不支持的操作系统")
}
