package main

import (
	"github.com/spf13/cobra"
)

var (
	syncDatabaseURL string
)

func init() {
	syncCmd.PersistentFlags().StringVarP(&syncDatabaseURL, "url", "u", "postgres://postgres@localhost:5432/?sslmode=disable", "database url")
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync contentful data",
}
