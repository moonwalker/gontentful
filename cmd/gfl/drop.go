package main

import (
	"github.com/spf13/cobra"
)

var (
	dropDatabaseURL string
	dropSchema      string
)

func init() {
	dropCmd.PersistentFlags().StringVarP(&dropDatabaseURL, "url", "u", "", "database url")
	if dropDatabaseURL == "" {
		dropDatabaseURL = "postgres://postgres@localhost:5432/?sslmode=disable"
	}
	dropCmd.PersistentFlags().StringVarP(&dropSchema, "schema", "n", "", "schema name")
	if dropSchema == "" {
		dropSchema = schemaName
	}
	rootCmd.AddCommand(dropCmd)
}

var dropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop schema",
}
