package cmd

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNewMysqlrestoreCmd_Basic(t *testing.T) {
	cmd := NewMysqlrestoreCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "mysqlrestore")
}

func TestNewMysqlrestoreCmd_HasFlags(t *testing.T) {
	cmd := NewMysqlrestoreCmd()
	assert.NotNil(t, cmd.Flags().Lookup("input"))
}

func TestNewMysqlrestoreCmd_NoArgsReturnsHelp(t *testing.T) {
	cmd := NewMysqlrestoreCmd()
	cmd.SetArgs([]string{"-h", "localhost", "-u", "root"})
	assert.NoError(t, cmd.Execute())
}

