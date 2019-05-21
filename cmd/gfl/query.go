package main

import (
	"github.com/spf13/cobra"
)

const (
	defaultLocale = "en"
	usePreview    = false
)

var (
	queryDatabaseURL string
)

func init() {
	queryCmd.PersistentFlags().StringVarP(&queryDatabaseURL, "url", "u", "", "database url")
	if queryDatabaseURL == "" {
		queryDatabaseURL = "postgres://postgres@localhost:5432/?sslmode=disable"
	}
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Generates a query from contentful request",
}
