package main

import (
	"github.com/spf13/cobra"
)

const (
	defaultLocale = "en"
)

var (
	queryDatabaseURL string
)

func init() {
	queryCmd.PersistentFlags().StringVarP(&queryDatabaseURL, "url", "u", "", "database url")
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Generates a query from contentful request",
}
