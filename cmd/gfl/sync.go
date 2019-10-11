package main

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync contentful data",
}
