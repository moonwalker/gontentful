package main

import (
	"github.com/spf13/cobra"
)

var (
	dropSchema      string
)

func init() {
	dropCmd.PersistentFlags().StringVarP(&dropSchema, "drop", "d", "", "schema to drop")
	if dropSchema == "" {
		dropSchema = schemaName
	}
	rootCmd.AddCommand(dropCmd)
}

var dropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop schema",
}
