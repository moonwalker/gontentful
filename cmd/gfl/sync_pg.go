package main

import (
	"log"

	"github.com/moonwalker/gontentful"
	"github.com/spf13/cobra"
)

var (
	initSync bool
)

func init() {
	pgSyncCmd.PersistentFlags().BoolVarP(&initSync, "init", "i", false, "init sync")
	syncCmd.AddCommand(pgSyncCmd)
}

var pgSyncCmd = &cobra.Command{
	Use:   "pg",
	Short: "Sync data to postgres",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		log.Println("get types...")
		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get types done")

		var syncToken string
		if !initSync {
			// TODO: try to get the token from db if exists
		}

		log.Println("sync...")
		res, err := client.Spaces.Sync(syncToken)
		log.Println("sync done")
		if err != nil {
			log.Fatal(err)
		}

		log.Println("bulk insert...")
		schema := gontentful.NewPGSyncSchema(schemaName, assetTableName, types.Items, res.Items)
		err = schema.BulkInsert(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("bulk insert done")

		// TODO: store token to db
	},
}
