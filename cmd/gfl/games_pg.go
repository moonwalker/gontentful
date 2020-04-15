package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

func init() {
	gamesCmd.AddCommand(pgGamesCmd)
}

var pgGamesCmd = &cobra.Command{
	Use:   "pg",
	Short: "Create games postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("creating %s games schema...", schemaName)
		query := gontentful.NewPGGames(schemaName)
		err := query.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Printf("%s games schema created successfully", schemaName)
	},
}
