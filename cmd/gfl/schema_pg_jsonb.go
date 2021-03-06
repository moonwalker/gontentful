package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

func init() {
	schemaCmd.AddCommand(jsonbSchemaCmd)
}

var jsonbSchemaCmd = &cobra.Command{
	Use:   "jsonb",
	Short: "Creates postgres jsonb schema",

	Run: func(cmd *cobra.Command, args []string) {
		if len(databaseURL) > 0 {
			log.Println("creating postgres jsonb schema...")
		}

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:        apiURL,
			SpaceID:       spaceID,
			EnvironmentID: environmentID,
			CdnToken:      cdnToken,
		})

		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}

		schema := gontentful.NewPGJSONBSchema(schemaName, types.Items)
		str, err := schema.Render()
		if err != nil {
			log.Fatal(err)
		}

		if len(databaseURL) == 0 {
			fmt.Println(str)
			return
		} else {
			log.Println("postgres jsonb schema successfully created")
		}
	},
}
