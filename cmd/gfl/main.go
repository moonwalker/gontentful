package main

import (
	"fmt"
	"os"
)

var (
	SpaceId string
	CdnToken string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&SpaceId, "space", "s", "", "cf space id (required)")
	rootCmd.PersistentFlags().StringVarP(&CdnToken, "cdn", "c", "", "cdn token (required)")
	rootCmd.MarkFlagRequired("space")
	rootCmd.MarkFlagRequired("cdn")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
