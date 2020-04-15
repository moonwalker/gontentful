package main

import (
	"github.com/spf13/cobra"
)

func init() {
	gamesCmd.MarkFlagRequired("schema")
	rootCmd.AddCommand(gamesCmd)
}

var gamesCmd = &cobra.Command{
	Use:   "games",
	Short: "Create games schema",
}
