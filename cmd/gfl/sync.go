package main

import (
	"github.com/spf13/cobra"
)

func init() {
	syncCmd.PersistentFlags().StringVarP(&databaseURL, "url", "u", "", "database url")
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync contentful data",
}
