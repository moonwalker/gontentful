package main

import (
	"fmt"
	"os"

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
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schema := gontentful.NewGraphQLSchema(types.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(str)
	},
}
