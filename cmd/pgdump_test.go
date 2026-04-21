package cmd

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewPgdumpCmd_Basic(t *testing.T) {
	cmd := NewPgdumpCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "pgdump")
}

func TestNewPgdumpCmd_HasFlags(t *testing.T) {
	cmd := NewPgdumpCmd()
	for _, f := range []string{"host", "port", "user", "password", "database", "output", "format"} {
		assert.NotNil(t, cmd.Flags().Lookup(f))
	}
}

func TestNewPgdumpCmd_FlagDefaults(t *testing.T) {
	cmd := NewPgdumpCmd()
	host, _ := cmd.Flags().GetString("host")
	assert.Equal(t, "127.0.0.1", host)
	port, _ := cmd.Flags().GetInt("port")
	assert.Equal(t, 5432, port)
	user, _ := cmd.Flags().GetString("user")
	assert.Equal(t, "postgres", user)
}

func TestNewPgdumpCmd_NoDatabaseReturnsHelp(t *testing.T) {
	cmd := NewPgdumpCmd()
	cmd.SetArgs([]string{"-h", "localhost"})
	assert.NoError(t, cmd.Execute())
}

