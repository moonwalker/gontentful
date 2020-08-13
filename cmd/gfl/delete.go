package main

import (
	"github.com/spf13/cobra"
)

func init() {
	deleteCmd.MarkFlagRequired("schema")
	rootCmd.AddCommand(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "del",
	Short: "Delete content",
}
