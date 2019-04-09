package main

import (
	"fmt"
	"log"
	"os"

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
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schema := gontentful.NewPGJSONBSchema(schemaName, assetTableName, types.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(databaseURL) == 0 {
			fmt.Println(str)
			return
		} else {
			log.Println("postgres jsonb schema successfully created")
		}
	},
}
