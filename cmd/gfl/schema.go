package main

import (
	"github.com/spf13/cobra"
)

const (
	schemaName     = "content"
	assetTableName = "_assets"
)

var (
	databaseURL string
)

func init() {
	schemaCmd.PersistentFlags().StringVarP(&databaseURL, "url", "u", "", "database url")
	rootCmd.AddCommand(schemaCmd)
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Creates schema from contentful types",
}
