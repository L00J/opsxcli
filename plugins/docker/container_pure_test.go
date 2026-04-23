package docker

import (
	"encoding/json"
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// formatContainerName 纯函数测试
// =============================================================================

func TestFormatContainerName_Normal(t *testing.T) {
	assert.Equal(t, "web-server", formatContainerName([]string{"/web-server"}))
}

func TestFormatContainerName_NoSlash(t *testing.T) {
	assert.Equal(t, "mycontainer", formatContainerName([]string{"mycontainer"}))
}

func TestFormatContainerName_Empty(t *testing.T) {
	assert.Equal(t, "", formatContainerName([]string{}))
}

func TestFormatContainerName_Nil(t *testing.T) {
	assert.Equal(t, "", formatContainerName(nil))
}

// =============================================================================
// formatPorts 纯函数测试
// =============================================================================

func TestFormatPorts_WithPublicPort(t *testing.T) {
	ports := []PortBinding{
		{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
	}
	result := formatPorts(ports)
	assert.Equal(t, "0.0.0.0:8080->80/tcp", result)
}

func TestFormatPorts_WithoutPublicPort(t *testing.T) {
	ports := []PortBinding{
		{PrivatePort: 443, Type: "tcp"},
	}
	result := formatPorts(ports)
	assert.Equal(t, "443/tcp", result)
}

func TestFormatPorts_Multiple(t *testing.T) {
	ports := []PortBinding{
		{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
		{PrivatePort: 443, Type: "tcp"},
	}
	result := formatPorts(ports)
	assert.Contains(t, result, "0.0.0.0:8080->80/tcp")
	assert.Contains(t, result, "443/tcp")
}

func TestFormatPorts_Empty(t *testing.T) {
	assert.Equal(t, "", formatPorts(nil))
	assert.Equal(t, "", formatPorts([]PortBinding{}))
}

// =============================================================================
// formatContainerState 纯函数测试
// =============================================================================

func TestFormatContainerState_Running(t *testing.T) {
	assert.Equal(t, "Up", formatContainerState(ContainerState{Running: true}))
}

func TestFormatContainerState_Paused(t *testing.T) {
	assert.Equal(t, "Paused", formatContainerState(ContainerState{Running: true, Paused: true}))
}

func TestFormatContainerState_Restarting(t *testing.T) {
	assert.Equal(t, "Restarting", formatContainerState(ContainerState{Restarting: true}))
}

func TestFormatContainerState_Exited(t *testing.T) {
	assert.Equal(t, "Exited (137)", formatContainerState(ContainerState{ExitCode: 137}))
}

func TestFormatContainerState_ExitedZero(t *testing.T) {
	assert.Equal(t, "Exited (0)", formatContainerState(ContainerState{}))
}

// =============================================================================
// parseContainerInspect 纯函数测试
// =============================================================================

func TestParseContainerInspect_Full(t *testing.T) {
	raw := map[string]interface{}{
		"Id":   "abc123",
		"Name": "/my-container",
		"Config": map[string]interface{}{
			"Image": "nginx:latest",
			"Env":   []interface{}{"PATH=/usr/local/bin", "HOME=/root"},
			"Labels": map[string]interface{}{
				"com.docker.compose.service": "web",
			},
		},
		"Created": "2026-04-20T10:00:00Z",
		"State": map[string]interface{}{
			"Status":     "running",
			"Running":    true,
			"Paused":     false,
			"Restarting": false,
			"ExitCode":   float64(0),
			"StartedAt":  "2026-04-20T10:00:01Z",
			"FinishedAt": "0001-01-01T00:00:00Z",
		},
		"NetworkSettings": map[string]interface{}{
			"Gateway":   "172.17.0.1",
			"IPAddress": "172.17.0.2",
			"Networks": map[string]interface{}{
				"bridge": map[string]interface{}{},
			},
		},
		"Mounts": []interface{}{
			map[string]interface{}{
				"Source":      "/host/data",
				"Destination": "/container/data",
				"Mode":        "rw",
				"RW":          true,
				"Type":        "bind",
			},
		},
	}

	result := parseContainerInspect(raw)
	require.NotNil(t, result)
	assert.Equal(t, "abc123", result.ID)
	assert.Equal(t, "my-container", result.Name)
	assert.Equal(t, "nginx:latest", result.Image)
	assert.Len(t, result.Env, 2)
	assert.Contains(t, result.Env, "PATH=/usr/local/bin")
	assert.Equal(t, "web", result.Labels["com.docker.compose.service"])
	assert.True(t, result.State.Running)
	assert.Equal(t, "172.17.0.1", result.NetworkSettings.Gateway)
	assert.Equal(t, "172.17.0.2", result.NetworkSettings.IPAddress)
	assert.Contains(t, result.NetworkSettings.Networks, "bridge")
	assert.Len(t, result.Mounts, 1)
	assert.Equal(t, "/host/data", result.Mounts[0].Source)
	assert.Equal(t, "/container/data", result.Mounts[0].Destination)
	assert.True(t, result.Mounts[0].RW)
}

func TestParseContainerInspect_Minimal(t *testing.T) {
	raw := map[string]interface{}{}
	result := parseContainerInspect(raw)
	require.NotNil(t, result)
	assert.Equal(t, "", result.ID)
	assert.Equal(t, "", result.Name)
}

func TestParseContainerInspect_InvalidCreated(t *testing.T) {
	raw := map[string]interface{}{
		"Created": "not-a-date",
	}
	result := parseContainerInspect(raw)
	require.NotNil(t, result)
	assert.True(t, result.Created.IsZero())
}

func TestParseContainerInspect_HealthStatus(t *testing.T) {
	raw := map[string]interface{}{
		"State": map[string]interface{}{
			"Status":  "running",
			"Running": true,
			"Health": map[string]interface{}{
				"Status": "healthy",
			},
		},
	}
	result := parseContainerInspect(raw)
	require.NotNil(t, result)
	assert.Equal(t, "healthy", result.State.Health)
}

// =============================================================================
// parseContainerState 纯函数测试
// =============================================================================

func TestParseContainerState_Full(t *testing.T) {
	raw := map[string]interface{}{
		"Status":     "running",
		"Running":    true,
		"Paused":     false,
		"Restarting": false,
		"ExitCode":   float64(0),
		"StartedAt":  "2026-04-20T10:00:01Z",
		"FinishedAt": "0001-01-01T00:00:00Z",
		"Health": map[string]interface{}{
			"Status": "healthy",
		},
	}
	state := parseContainerState(raw)
	assert.Equal(t, "running", state.Status)
	assert.True(t, state.Running)
	assert.False(t, state.Paused)
	assert.Equal(t, 0, state.ExitCode)
	assert.Equal(t, "healthy", state.Health)
}

func TestParseContainerState_Empty(t *testing.T) {
	raw := map[string]interface{}{}
	state := parseContainerState(raw)
	assert.False(t, state.Running)
	assert.Equal(t, 0, state.ExitCode)
}

func TestParseContainerState_ExitCodeFromInt(t *testing.T) {
	raw := map[string]interface{}{
		"ExitCode": 137,
	}
	state := parseContainerState(raw)
	assert.Equal(t, 137, state.ExitCode)
}

func TestParseContainerState_ExitCodeFromJSONNumber(t *testing.T) {
	raw := map[string]interface{}{
		"ExitCode": json.Number("42"),
	}
	state := parseContainerState(raw)
	assert.Equal(t, 42, state.ExitCode)
}

// =============================================================================
// parseNetworkSettings 纯函数测试
// =============================================================================

func TestParseNetworkSettings_Full(t *testing.T) {
	raw := map[string]interface{}{
		"Gateway":   "172.17.0.1",
		"IPAddress": "172.17.0.5",
		"Networks": map[string]interface{}{
			"bridge": map[string]interface{}{},
			"my-net": map[string]interface{}{},
		},
		"Ports": map[string]interface{}{
			"80/tcp": []interface{}{
				map[string]interface{}{
					"HostIp":   "0.0.0.0",
					"HostPort": "8080",
				},
			},
		},
	}
	ns := parseNetworkSettings(raw)
	assert.Equal(t, "172.17.0.1", ns.Gateway)
	assert.Equal(t, "172.17.0.5", ns.IPAddress)
	assert.Len(t, ns.Networks, 2)
	assert.Contains(t, ns.Networks, "bridge")
	assert.Contains(t, ns.Networks, "my-net")
	assert.Len(t, ns.Ports, 1)
	assert.Contains(t, ns.Ports[0], "0.0.0.0:8080->80/tcp")
}

func TestParseNetworkSettings_PortWithoutBinding(t *testing.T) {
	raw := map[string]interface{}{
		"Ports": map[string]interface{}{
			"443/tcp": []interface{}{},
		},
	}
	ns := parseNetworkSettings(raw)
	assert.Contains(t, ns.Ports[0], "443/tcp")
}

func TestParseNetworkSettings_Empty(t *testing.T) {
	ns := parseNetworkSettings(map[string]interface{}{})
	assert.Equal(t, "", ns.Gateway)
	assert.Equal(t, "", ns.IPAddress)
}

// =============================================================================
// parseMounts 纯函数测试
// =============================================================================

func TestParseMounts_Full(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"Source":      "/host/data",
			"Destination": "/data",
			"Mode":        "rw",
			"RW":          true,
			"Type":        "bind",
		},
		map[string]interface{}{
			"Source":      "myvolume",
			"Destination": "/app/data",
			"Mode":        "ro",
			"RW":          false,
			"Type":        "volume",
		},
	}
	mounts := parseMounts(raw)
	require.Len(t, mounts, 2)
	assert.Equal(t, "/host/data", mounts[0].Source)
	assert.Equal(t, "/data", mounts[0].Destination)
	assert.True(t, mounts[0].RW)
	assert.Equal(t, "bind", mounts[0].Type)
	assert.False(t, mounts[1].RW)
	assert.Equal(t, "volume", mounts[1].Type)
}

func TestParseMounts_Empty(t *testing.T) {
	mounts := parseMounts([]interface{}{})
	assert.Len(t, mounts, 0)
}

func TestParseMounts_Nil(t *testing.T) {
	mounts := parseMounts(nil)
	assert.Len(t, mounts, 0)
}

func TestParseMounts_PartialFields(t *testing.T) {
	raw := []interface{}{
		map[string]interface{}{
			"Source": "/src",
			// 其他字段缺失
		},
	}
	mounts := parseMounts(raw)
	require.Len(t, mounts, 1)
	assert.Equal(t, "/src", mounts[0].Source)
	assert.Equal(t, "", mounts[0].Destination)
	assert.False(t, mounts[0].RW)
}

// =============================================================================
// toInt 纯函数测试
// =============================================================================

func TestToInt_Float64(t *testing.T) {
	val, ok := toInt(float64(42))
	assert.True(t, ok)
	assert.Equal(t, 42, val)
}

func TestToInt_Int(t *testing.T) {
	val, ok := toInt(137)
	assert.True(t, ok)
	assert.Equal(t, 137, val)
}

func TestToInt_JSONNumber(t *testing.T) {
	val, ok := toInt(json.Number("255"))
	assert.True(t, ok)
	assert.Equal(t, 255, val)
}

func TestToInt_InvalidJSONNumber(t *testing.T) {
	_, ok := toInt(json.Number("not-a-number"))
	assert.False(t, ok)
}

func TestToInt_String(t *testing.T) {
	_, ok := toInt("42")
	assert.False(t, ok)
}

func TestToInt_Nil(t *testing.T) {
	_, ok := toInt(nil)
	assert.False(t, ok)
}

func TestToInt_NegativeFloat(t *testing.T) {
	val, ok := toInt(float64(-1))
	assert.True(t, ok)
	assert.Equal(t, -1, val)
}

// =============================================================================
// cleanDockerError 纯函数测试
// =============================================================================

func TestCleanDockerError_ExitError(t *testing.T) {
	err := &exec.ExitError{Stderr: []byte("  container not found  ")}
	result := cleanDockerError(err)
	assert.Equal(t, "container not found", result)
}

func TestCleanDockerError_ExitErrorEmptyStderr(t *testing.T) {
	err := &exec.ExitError{Stderr: []byte("")}
	result := cleanDockerError(err)
	assert.NotEmpty(t, result) // 返回 err.Error()
}

func TestCleanDockerError_GenericError(t *testing.T) {
	err := errors.New("something went wrong")
	result := cleanDockerError(err)
	assert.Equal(t, "something went wrong", result)
}

// =============================================================================
// PrintInspect 格式化输出测试（不验证 stdout，只验证不 panic）
// =============================================================================

func TestPrintInspect_NoPanic(t *testing.T) {
	result := &ContainerInspectResult{
		ID:    "abc123",
		Name:  "test-container",
		Image: "nginx:latest",
		Created: func() time.Time {
			tm, _ := time.Parse(time.RFC3339, "2026-04-20T10:00:00Z")
			return tm
		}(),
		State: ContainerState{
			Status:     "running",
			Running:    true,
			ExitCode:   0,
			StartedAt:  "2026-04-20T10:00:01Z",
			FinishedAt: "",
			Health:     "healthy",
		},
		NetworkSettings: NetworkSummary{
			IPAddress: "172.17.0.2",
			Gateway:   "172.17.0.1",
			Networks:  []string{"bridge"},
			Ports:     []string{"0.0.0.0:8080->80/tcp"},
		},
		Labels: map[string]string{"app": "web"},
		Env:    []string{"PATH=/usr/local/bin"},
		Mounts: []MountInfo{
			{Source: "/host", Destination: "/data", Mode: "rw", RW: true, Type: "bind"},
		},
	}
	// 仅验证不 panic
	assert.NotPanics(t, func() { PrintInspect(result) })
}
