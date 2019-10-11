package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	spaceID  string
	cdnToken string
	cmaToken string
	databaseURL string
)

const (
	apiURL        = "cdn.contentful.com"
	cmaURL        = "api.contentful.com"
	schemaName    = "content"
	defaultLocale = "en"
)

var rootCmd = &cobra.Command{
	Use:   "gontentful",
	Short: "cli for contentful",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&spaceID, "space", "s", "", "cf space id (required)")
	rootCmd.PersistentFlags().StringVarP(&cdnToken, "token", "t", "", "cdn token (required)")
	rootCmd.PersistentFlags().StringVarP(&cmaToken, "cma", "c", "", "cma token (required)")
	rootCmd.PersistentFlags().StringVarP(&databaseURL, "url", "u", "postgres://postgres@localhost:5432/?sslmode=disable", "database url")
	rootCmd.MarkFlagRequired("space")
	rootCmd.MarkFlagRequired("token")
	rootCmd.MarkFlagRequired("cma")
	rootCmd.MarkFlagRequired("url")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
