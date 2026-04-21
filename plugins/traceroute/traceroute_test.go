package traceroute

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Constants ---

func TestDefaultMaxHops(t *testing.T) {
	assert.Equal(t, 30, DefaultMaxHops)
}

func TestDefaultPacketSize(t *testing.T) {
	assert.Equal(t, 60, DefaultPacketSize)
}

func TestDefaultTimeout(t *testing.T) {
	assert.Equal(t, 3*time.Second, DefaultTimeout)
}

func TestDefaultPort(t *testing.T) {
	assert.Equal(t, 33434, DefaultPort)
}

// Note: Traceroute function cannot be easily unit-tested because:
// 1. It requires raw socket privileges (ICMP/UDP)
// 2. Invalid hosts cause logger mutex deadlock
// Traceroute should be tested via integration tests with valid hosts.
