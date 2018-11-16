package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

const (
	apiURL = "api.contentful.com"
)

func init() {
	schemaCmd.AddCommand(pgSchemaCmd)
	schemaCmd.AddCommand(gqlSchemaCmd)
	rootCmd.AddCommand(schemaCmd)
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Creates schema from contentful types",
}

var pgSchemaCmd = &cobra.Command{
	Use:   "pg",
	Short: "Creates postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		data, err := client.ContentTypes.Get()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		resp := &gontentful.ContentTypes{}
		err = json.Unmarshal(data, resp)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schema := gontentful.NewPGSQLSchema(SpaceId, resp.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(str)
	},
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

		data, err := client.ContentTypes.Get()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		resp := &gontentful.ContentTypes{}
		err = json.Unmarshal(data, resp)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schema := gontentful.NewGraphQLSchema(resp.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(str)
	},
}
