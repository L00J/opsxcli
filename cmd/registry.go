package cmd

import "github.com/spf13/cobra"

type CommandEntry struct {
	Category    string
	Description string
	Factory     func() *cobra.Command
}

var commandRegistry = make(map[string]CommandEntry)

func RegisterCommand(name string, category string, description string, factory func() *cobra.Command) {
	commandRegistry[name] = CommandEntry{Category: category, Description: description, Factory: factory}
}

func GetRegisteredCommands() map[string]CommandEntry {
	return commandRegistry
}
