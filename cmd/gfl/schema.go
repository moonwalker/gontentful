package main

import (
	"github.com/spf13/cobra"
)

const (
	schemaName = "content"
)

var (
	schemaDatabaseURL string
)

func init() {
	schemaCmd.PersistentFlags().StringVarP(&schemaDatabaseURL, "url", "u", "", "database url")
	rootCmd.AddCommand(schemaCmd)
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Creates schema from contentful types",
}
