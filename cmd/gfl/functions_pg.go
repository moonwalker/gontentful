package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

func init() {
	funcCmd.AddCommand(pgFuncCmd)
}

var pgFuncCmd = &cobra.Command{
	Use:   "pg",
	Short: "Create or replace postgres functions",

	Run: func(cmd *cobra.Command, args []string) {
		log.Println("creating or replacing functions...")
		schema := gontentful.NewPGFunctions(schemaName)
		err := schema.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("exec done")
	},
}
