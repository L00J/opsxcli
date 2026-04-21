package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestRSAKey creates a temporary RSA private key file for testing
func generateTestRSAKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	})

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_rsa_key")
	err = os.WriteFile(keyPath, keyPEM, 0600)
	require.NoError(t, err)

	return keyPath
}

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

func TestLoadPrivateKey_NonexistentFile(t *testing.T) {
	_, err := loadPrivateKey("/tmp/nonexistent_key_12345")
	assert.Error(t, err)
}

func TestLoadPrivateKey_InvalidKeyData(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "bad_key")
	err := os.WriteFile(keyPath, []byte("this is not a valid private key"), 0600)
	require.NoError(t, err)

	_, err = loadPrivateKey(keyPath)
	assert.Error(t, err)
}

func TestLoadPrivateKey_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "empty_key")
	err := os.WriteFile(keyPath, []byte(""), 0600)
	require.NoError(t, err)

	_, err = loadPrivateKey(keyPath)
	assert.Error(t, err)
}

func TestLoadPrivateKey_ValidRSAKey(t *testing.T) {
	keyPath := generateTestRSAKey(t)

	signer, err := loadPrivateKey(keyPath)
	require.NoError(t, err)
	require.NotNil(t, signer)
	assert.NotNil(t, signer.PublicKey())
}

func TestLoadPrivateKey_DirectoryPath(t *testing.T) {
	tmpDir := t.TempDir()
	// Loading a directory should fail
	_, err := loadPrivateKey(tmpDir)
	assert.Error(t, err)
}

func TestLoadPrivateKey_NonPEMData(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "binary_key")
	// Write random binary data
	err := os.WriteFile(keyPath, []byte{0x00, 0x01, 0x02, 0x03, 0xFF}, 0600)
	require.NoError(t, err)

	_, err = loadPrivateKey(keyPath)
	assert.Error(t, err)
}
