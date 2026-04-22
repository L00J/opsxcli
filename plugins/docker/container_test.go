package docker

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// === formatContainerName ===

func TestFormatContainerName(t *testing.T) {
	tests := []struct {
		name     string
		names    []string
		expected string
	}{
		{"single name", []string{"/nginx"}, "nginx"},
		{"name without slash", []string{"nginx"}, "nginx"},
		{"empty names", []string{}, ""},
		{"nil names", nil, ""},
		{"first name selected", []string{"/web", "/web-1"}, "web"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatContainerName(tt.names)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// === formatPorts ===

func TestFormatPorts(t *testing.T) {
	tests := []struct {
		name     string
		ports    []PortBinding
		expected string
	}{
		{
			"nil ports",
			nil,
			"",
		},
		{
			"empty ports",
			[]PortBinding{},
			"",
		},
		{
			"public port mapping",
			[]PortBinding{{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"}},
			"0.0.0.0:8080->80/tcp",
		},
		{
			"private port only",
			[]PortBinding{{PrivatePort: 443, Type: "tcp"}},
			"443/tcp",
		},
		{
			"multiple ports",
			[]PortBinding{
				{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
				{IP: "0.0.0.0", PrivatePort: 443, PublicPort: 8443, Type: "tcp"},
			},
			"0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPorts(tt.ports)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// === formatContainerState ===

func TestFormatContainerState(t *testing.T) {
	tests := []struct {
		name     string
		state    ContainerState
		expected string
	}{
		{
			"running",
			ContainerState{Running: true},
			"Up",
		},
		{
			"paused",
			ContainerState{Running: true, Paused: true},
			"Paused",
		},
		{
			"restarting",
			ContainerState{Restarting: true},
			"Restarting",
		},
		{
			"exited with code 0",
			ContainerState{ExitCode: 0},
			"Exited (0)",
		},
		{
			"exited with code 137",
			ContainerState{ExitCode: 137},
			"Exited (137)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatContainerState(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// === parseContainerInspect ===

func TestParseContainerInspect(t *testing.T) {
	t.Run("basic fields", func(t *testing.T) {
		raw := map[string]interface{}{
			"Id":   "abc123def456",
			"Name": "/my-container",
			"Config": map[string]interface{}{
				"Image": "nginx:latest",
				"Env":   []interface{}{"PATH=/usr/local/bin", "NODE_ENV=production"},
				"Labels": map[string]interface{}{
					"com.docker.compose.service": "web",
				},
			},
			"Created": "2024-01-15T10:30:00Z",
			"State": map[string]interface{}{
				"Status":     "running",
				"Running":    true,
				"Paused":     false,
				"Restarting": false,
				"ExitCode":   float64(0),
				"StartedAt":  "2024-01-15T10:30:01Z",
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

		assert.Equal(t, "abc123def456", result.ID)
		assert.Equal(t, "my-container", result.Name)
		assert.Equal(t, "nginx:latest", result.Image)
		assert.True(t, result.State.Running)
		assert.False(t, result.State.Paused)
		assert.Equal(t, 0, result.State.ExitCode)
		assert.Equal(t, "172.17.0.1", result.NetworkSettings.Gateway)
		assert.Equal(t, "172.17.0.2", result.NetworkSettings.IPAddress)
		assert.Contains(t, result.NetworkSettings.Networks, "bridge")
		assert.Len(t, result.Mounts, 1)
		assert.Equal(t, "/host/data", result.Mounts[0].Source)
		assert.Equal(t, "/container/data", result.Mounts[0].Destination)
		assert.True(t, result.Mounts[0].RW)
		assert.Equal(t, "bind", result.Mounts[0].Type)
		assert.Len(t, result.Env, 2)
		assert.Contains(t, result.Env, "PATH=/usr/local/bin")
		assert.Equal(t, "web", result.Labels["com.docker.compose.service"])
	})

	t.Run("with health check", func(t *testing.T) {
		raw := map[string]interface{}{
			"Id":   "health-container",
			"Name": "/health-check",
			"Config": map[string]interface{}{
				"Image": "redis:alpine",
			},
			"State": map[string]interface{}{
				"Status":  "running",
				"Running": true,
				"Health": map[string]interface{}{
					"Status": "healthy",
				},
			},
			"NetworkSettings": map[string]interface{}{},
		}

		result := parseContainerInspect(raw)
		assert.Equal(t, "healthy", result.State.Health)
	})

	t.Run("empty fields", func(t *testing.T) {
		raw := map[string]interface{}{}
		result := parseContainerInspect(raw)
		assert.Empty(t, result.ID)
		assert.Empty(t, result.Name)
		assert.Empty(t, result.Image)
	})
}

// === parseContainerState ===

func TestParseContainerState(t *testing.T) {
	t.Run("with all fields", func(t *testing.T) {
		raw := map[string]interface{}{
			"Status":      "running",
			"Running":     true,
			"Paused":      false,
			"Restarting":  false,
			"ExitCode":    float64(0),
			"StartedAt":   "2024-01-15T10:30:01Z",
			"FinishedAt":  "2024-01-14T08:00:00Z",
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
	})

	t.Run("minimal fields", func(t *testing.T) {
		raw := map[string]interface{}{}
		state := parseContainerState(raw)
		assert.Empty(t, state.Status)
		assert.False(t, state.Running)
		assert.Equal(t, 0, state.ExitCode)
	})
}

// === parseNetworkSettings ===

func TestParseNetworkSettings(t *testing.T) {
	t.Run("with port bindings", func(t *testing.T) {
		raw := map[string]interface{}{
			"Gateway":   "172.17.0.1",
			"IPAddress": "172.17.0.3",
			"Networks": map[string]interface{}{
				"bridge": map[string]interface{}{},
				"host":   map[string]interface{}{},
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
		assert.Equal(t, "172.17.0.3", ns.IPAddress)
		assert.Len(t, ns.Networks, 2)
		assert.Contains(t, ns.Networks, "bridge")
		assert.Contains(t, ns.Networks, "host")
		assert.Len(t, ns.Ports, 1)
		assert.Contains(t, ns.Ports[0], "8080")
		assert.Contains(t, ns.Ports[0], "80/tcp")
	})

	t.Run("exposed port without binding", func(t *testing.T) {
		raw := map[string]interface{}{
			"Ports": map[string]interface{}{
				"443/tcp": nil,
			},
		}
		ns := parseNetworkSettings(raw)
		assert.Len(t, ns.Ports, 1)
		assert.Equal(t, "443/tcp", ns.Ports[0])
	})

	t.Run("empty settings", func(t *testing.T) {
		raw := map[string]interface{}{}
		ns := parseNetworkSettings(raw)
		assert.Empty(t, ns.Gateway)
		assert.Empty(t, ns.IPAddress)
	})
}

// === parseMounts ===

func TestParseMounts(t *testing.T) {
	t.Run("multiple mounts", func(t *testing.T) {
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
				"Destination": "/var/lib/data",
				"Mode":        "z",
				"RW":          true,
				"Type":        "volume",
			},
		}
		mounts := parseMounts(raw)
		assert.Len(t, mounts, 2)
		assert.Equal(t, "bind", mounts[0].Type)
		assert.True(t, mounts[0].RW)
		assert.Equal(t, "volume", mounts[1].Type)
	})

	t.Run("empty mounts", func(t *testing.T) {
		mounts := parseMounts([]interface{}{})
		assert.Empty(t, mounts)
	})

	t.Run("nil mounts", func(t *testing.T) {
		mounts := parseMounts(nil)
		assert.Empty(t, mounts)
	})
}

// === toInt ===

func TestToInt(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expected  int
		expectOk  bool
	}{
		{"float64", float64(42), 42, true},
		{"int", 42, 42, true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := toInt(tt.input)
			assert.Equal(t, tt.expectOk, ok)
			if ok {
				assert.Equal(t, tt.expected, val)
			}
		})
	}
}

// === cleanDockerError ===

func TestCleanDockerError(t *testing.T) {
	t.Run("generic error", func(t *testing.T) {
		err := cleanDockerError(assert.AnError)
		assert.Contains(t, err, "assert")
	})
}

// === ContainerInfo JSON parsing ===

func TestContainerInfoJSON(t *testing.T) {
	raw := `{
		"Id": "abc123",
		"Names": ["/my-container"],
		"Image": "nginx:latest",
		"State": "running",
		"Status": "Up 2 hours",
		"Ports": [
			{"IP": "0.0.0.0", "PrivatePort": 80, "PublicPort": 8080, "Type": "tcp"}
		]
	}`

	var c ContainerInfo
	err := json.Unmarshal([]byte(raw), &c)
	assert.NoError(t, err)
	assert.Equal(t, "abc123", c.ID)
	assert.Equal(t, []string{"/my-container"}, c.Names)
	assert.Equal(t, "nginx:latest", c.Image)
	assert.Equal(t, "running", c.State)
	assert.Len(t, c.Ports, 1)
	assert.Equal(t, 80, c.Ports[0].PrivatePort)
}

// === PSOptions default values ===

func TestPSOptionsDefaults(t *testing.T) {
	opts := &PSOptions{}
	assert.False(t, opts.All)
	assert.Equal(t, 0, opts.Last)
	assert.Empty(t, opts.Filter)
	assert.Empty(t, opts.Format)
	assert.False(t, opts.NoTrunc)
	assert.False(t, opts.Quiet)
}

// === ContainerActionOptions defaults ===

func TestContainerActionOptionsDefaults(t *testing.T) {
	opts := &ContainerActionOptions{}
	assert.Empty(t, opts.Containers)
	assert.Equal(t, 0, opts.Timeout)
	assert.False(t, opts.Force)
	assert.False(t, opts.Volumes)
}

// === PortBinding ===

func TestPortBinding(t *testing.T) {
	pb := PortBinding{
		IP:          "0.0.0.0",
		PrivatePort: 80,
		PublicPort:  8080,
		Type:        "tcp",
	}
	assert.Equal(t, "0.0.0.0", pb.IP)
	assert.Equal(t, 80, pb.PrivatePort)
	assert.Equal(t, 8080, pb.PublicPort)
	assert.Equal(t, "tcp", pb.Type)
}

// === ContainerState ===

func TestContainerStateDefaults(t *testing.T) {
	state := ContainerState{}
	assert.Empty(t, state.Status)
	assert.False(t, state.Running)
	assert.False(t, state.Paused)
	assert.False(t, state.Restarting)
	assert.Equal(t, 0, state.ExitCode)
	assert.Empty(t, state.Health)
}

// === NetworkSummary ===

func TestNetworkSummaryDefaults(t *testing.T) {
	ns := NetworkSummary{}
	assert.Empty(t, ns.IPAddress)
	assert.Empty(t, ns.Gateway)
	assert.Empty(t, ns.Networks)
	assert.Empty(t, ns.Ports)
}

// === MountInfo ===

func TestMountInfo(t *testing.T) {
	m := MountInfo{
		Source:      "/host/path",
		Destination: "/container/path",
		Mode:        "rw",
		RW:          true,
		Type:        "bind",
	}
	assert.Equal(t, "/host/path", m.Source)
	assert.Equal(t, "/container/path", m.Destination)
	assert.True(t, m.RW)
}

// === ContainerInspectResult ===

func TestContainerInspectResultDefaults(t *testing.T) {
	result := ContainerInspectResult{}
	assert.Empty(t, result.ID)
	assert.Empty(t, result.Name)
	assert.Empty(t, result.Image)
	assert.Empty(t, result.Env)
	assert.Empty(t, result.Mounts)
	assert.Nil(t, result.Labels)
}

// === Container lifecycle errors (no docker) ===

func TestPS_NoDocker(t *testing.T) {
	// 测试无 docker 时的错误处理
	// 这个测试在无 docker 环境下会返回错误
	err := PS(&PSOptions{})
	// 在 CI 环境可能没有 docker，所以只验证不会 panic
	_ = err
}

func TestStop_NoArgs(t *testing.T) {
	err := Stop(&ContainerActionOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")
}

func TestStart_NoArgs(t *testing.T) {
	err := Start(&ContainerActionOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")
}

func TestRestart_NoArgs(t *testing.T) {
	err := Restart(&ContainerActionOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")
}

func TestRM_NoArgs(t *testing.T) {
	err := RM(&ContainerActionOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "请指定")
}
