package main

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(funcCmd)
}

var funcCmd = &cobra.Command{
	Use:   "func",
	Short: "Created functions",
}
