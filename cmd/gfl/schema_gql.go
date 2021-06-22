package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

func init() {
	schemaCmd.AddCommand(gqlSchemaCmd)
}

var gqlSchemaCmd = &cobra.Command{
	Use:   "gql",
	Short: "Creates graphql schema",

	Run: func(cmd *cobra.Command, args []string) {
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

		schema := gontentful.NewGraphQLSchema(types.Items)
		str, err := schema.Render()
		if err != nil {
			log.Fatal(err)
		}

		if len(str) > 0 {
			fmt.Println(str)
		}
	},
}
