package main

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Generates a query from contentful request",
}
