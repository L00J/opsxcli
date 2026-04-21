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

// Note: DetectOS() and Install() are not unit-tested here because:
// - DetectOS() reads from /etc/os-release, filesystem-dependent
// - Install() calls exec.Command and requires root privileges
// These should be tested via integration tests.
