package cmd

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewPgrestoreCmd_Basic(t *testing.T) {
	cmd := NewPgrestoreCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "pgrestore")
}

func TestNewPgrestoreCmd_HasFlags(t *testing.T) {
	cmd := NewPgrestoreCmd()
	for _, f := range []string{"host", "port", "user", "password", "database", "input"} {
		assert.NotNil(t, cmd.Flags().Lookup(f))
	}
}

func TestNewPgrestoreCmd_NoArgsReturnsHelp(t *testing.T) {
	cmd := NewPgrestoreCmd()
	cmd.SetArgs([]string{"-h", "localhost"})
	assert.NoError(t, cmd.Execute())
}

