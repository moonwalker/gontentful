package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	name    = "gontentful"
	version = "0.1.0"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version info",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s v%s\n", name, version)
	},
}
