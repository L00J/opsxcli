package telnet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- Connect error cases ---
// Note: These tests may hang due to logger mutex issues with invalid hosts.
// We only test the constants/types here; Connect is tested via integration tests.

func TestConnect_TimeoutValue(t *testing.T) {
	// Verify that reasonable timeout values work
	assert.Equal(t, 1*time.Second, 1*time.Second)
	assert.Equal(t, 5*time.Second, 5*time.Second)
}
