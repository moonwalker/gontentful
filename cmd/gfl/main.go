package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	SpaceID  string
	CdnToken string
	CmaToken string
)

const (
	apiURL        = "cdn.contentful.com"
	cmaURL        = "api.contentful.com"
	schemaName    = "content"
	usePreview    = false
	defaultLocale = "en"
)

var rootCmd = &cobra.Command{
	Use:   "gontentful",
	Short: "cli for contentful",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&SpaceID, "space", "s", "", "cf space id (required)")
	rootCmd.PersistentFlags().StringVarP(&CdnToken, "token", "t", "", "cdn token (required)")
	rootCmd.PersistentFlags().StringVarP(&CmaToken, "cma", "c", "", "cma token (required)")
	rootCmd.MarkFlagRequired("space")
	rootCmd.MarkFlagRequired("token")
	rootCmd.MarkFlagRequired("cma")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
