package main

import (
	"github.com/spf13/cobra"
)

func init() {
	publishCmd.MarkFlagRequired("schema")
	rootCmd.AddCommand(publishCmd)
}

var publishCmd = &cobra.Command{
	Use:   "pub",
	Short: "Publish content",
}
