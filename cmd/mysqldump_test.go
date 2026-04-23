package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMysqldumpCmd_Basic(t *testing.T) {
	cmd := NewMysqldumpCmd()
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "mysqldump")
}

func TestNewMysqldumpCmd_HasFlags(t *testing.T) {
	cmd := NewMysqldumpCmd()
	for _, f := range []string{"output", "format", "tables", "ignore-tables", "no-data", "no-schema"} {
		assert.NotNil(t, cmd.Flags().Lookup(f))
	}
}

func TestNewMysqldumpCmd_FlagDefaults(t *testing.T) {
	cmd := NewMysqldumpCmd()
	format, _ := cmd.Flags().GetString("format")
	assert.Equal(t, "sql", format)
	noData, _ := cmd.Flags().GetBool("no-data")
	assert.False(t, noData)
}

func TestNewMysqldumpCmd_NoDatabaseReturnsHelp(t *testing.T) {
	cmd := NewMysqldumpCmd()
	cmd.SetArgs([]string{"-h", "localhost", "-u", "root"})
	assert.NoError(t, cmd.Execute())
}
