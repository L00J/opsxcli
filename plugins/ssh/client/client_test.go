package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSSHClient_MissingAuth(t *testing.T) {
	_, err := NewSSHClient("127.0.0.1", 22, "root", "", "")
	assert.Error(t, err)
}

func TestNewSSHClient_InvalidHost(t *testing.T) {
	_, err := NewSSHClient("256.256.256.256", 99999, "user", "", "password")
	assert.Error(t, err)
}

func TestSSHClient_Struct(t *testing.T) {
	client := &SSHClient{
		host: "example.com",
		port: 22,
	}
	assert.Equal(t, "example.com", client.host)
	assert.Equal(t, 22, client.port)
}

