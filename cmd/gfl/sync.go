package main

import (
	"github.com/spf13/cobra"
)

func init() {
	syncCmd.PersistentFlags().BoolVarP(&withEntries, "entries", "e", false, "create _entries table")
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync contentful data",
}
