package cmd

import (
	"github.com/spf13/cobra"
)

// list is now an alias for the root command (nsh)
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "Browse hosts (alias for nsh)",
	RunE:    rootCmd.RunE,
}

func init() {
	rootCmd.AddCommand(listCmd)
}
