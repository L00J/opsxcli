package cloudhost

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- HostEntry ---

func TestHostEntry(t *testing.T) {
	entry := HostEntry{Hostname: "web01", IP: "192.168.1.10"}
	assert.Equal(t, "web01", entry.Hostname)
	assert.Equal(t, "192.168.1.10", entry.IP)
}

// --- ConsulServiceRegistration ---

func TestConsulServiceRegistration(t *testing.T) {
	reg := ConsulServiceRegistration{
		ID:      "web01-app",
		Name:    "webapp",
		Address: "192.168.1.10",
		Port:    8080,
		Checks: []ConsulServiceCheck{
			{HTTP: "http://192.168.1.10:8080/health", Interval: "10s"},
		},
	}
	assert.Equal(t, "web01-app", reg.ID)
	assert.Equal(t, "webapp", reg.Name)
	assert.Equal(t, "http://192.168.1.10:8080/health", reg.Checks[0].HTTP)
}

// --- ConsulHealthCheckResult ---

func TestConsulHealthCheckResult(t *testing.T) {
	result := ConsulHealthCheckResult{ServiceID: "svc-001"}
	assert.Equal(t, "svc-001", result.ServiceID)
}

// --- NewCloudHostRegistry ---

func TestNewCloudHostRegistry(t *testing.T) {
	hosts := []HostEntry{
		{Hostname: "host1", IP: "10.0.0.1"},
		{Hostname: "host2", IP: "10.0.0.2"},
	}
	registry := NewCloudHostRegistry("http://consul:8500/", hosts, 8080, 9100, "/metrics")
	assert.NotNil(t, registry)
	assert.Equal(t, "http://consul:8500", registry.consulURL) // trailing slash trimmed
	assert.Equal(t, 8080, registry.appPort)
	assert.Equal(t, 9100, registry.nodeExpPort)
	assert.Equal(t, "/metrics", registry.metricsPath)
	assert.Len(t, registry.hosts, 2)
}

func TestNewCloudHostRegistry_EmptyHosts(t *testing.T) {
	registry := NewCloudHostRegistry("http://localhost:8500", nil, 80, 9100, "/metrics")
	assert.NotNil(t, registry)
	assert.Empty(t, registry.hosts)
}
