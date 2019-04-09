package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

func init() {
	schemaCmd.AddCommand(pgSchemaCmd)
}

var pgSchemaCmd = &cobra.Command{
	Use:   "pg",
	Short: "Creates postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		if len(databaseURL) > 0 {
			log.Println("creating postgres schema...")
		}

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}

		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}

		schema := gontentful.NewPGSQLSchema(schemaName, space, types.Items)
		str, err := schema.Render()
		if err != nil {
			log.Fatal(err)
		}

		if len(databaseURL) == 0 {
			fmt.Println(str)
			return
		} else {
			log.Println("postgres schema successfully created")
		}

		log.Println("executing postgres schema...")
		db, _ := sql.Open("postgres", databaseURL)
		txn, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}

		_, err = db.Exec(str)
		if err != nil {
			log.Fatal(err)
		}

		err = txn.Commit()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("postgres schema successfully executed")
	},
}
