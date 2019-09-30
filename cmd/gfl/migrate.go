package main

import (
	"github.com/spf13/cobra"
)

var (
	migrateDatabaseURL string
)

func init() {
	migrateCmd.PersistentFlags().StringVarP(&migrateDatabaseURL, "url", "u", "postgres://postgres@localhost:5432/?sslmode=disable", "database url")
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate contentful schema with data",
}
