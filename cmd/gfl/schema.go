package main

import (
	"github.com/spf13/cobra"
)

func init() {
	schemaCmd.PersistentFlags().BoolVarP(&withMetaData, "meta", "m", false, "create meta tables")
	schemaCmd.PersistentFlags().BoolVarP(&withEntries, "entries", "e", false, "create _entries table")
	rootCmd.AddCommand(schemaCmd)
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Creates schema from contentful types",
}
