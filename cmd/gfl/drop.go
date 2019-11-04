package main

import (
	"github.com/spf13/cobra"
)

func init() {
	dropCmd.MarkFlagRequired("schema")
	rootCmd.AddCommand(dropCmd)
}

var dropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Drop schema",
}
