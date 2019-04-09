package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	SpaceId  string
	CdnToken string
)

const (
	apiURL = "cdn.contentful.com"
)

var rootCmd = &cobra.Command{
	Use:   "gontentful",
	Short: "cli for contentful",
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&SpaceId, "space", "s", "", "cf space id (required)")
	rootCmd.PersistentFlags().StringVarP(&CdnToken, "token", "t", "", "token token (required)")
	rootCmd.MarkFlagRequired("space")
	rootCmd.MarkFlagRequired("token")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
