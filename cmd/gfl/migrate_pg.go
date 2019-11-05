package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

func init() {
	migrateCmd.AddCommand(pgMigrateCmd)
}

var pgMigrateCmd = &cobra.Command{
	Use:   "pg",
	Short: "Migrate postgres schema with data",

	Run: func(cmd *cobra.Command, args []string) {
		if len(migrateDatabaseURL) == 0 {
			log.Println("database url must be specified")
		}

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  spaceID,
			CdnToken: cdnToken,
			CmaURL:   cmaURL,
			CmaToken: cmaToken,
		})

		log.Println("get space...")
		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get space done")

		log.Println("get types...")
		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get types done")

		log.Println("get cma types...")
		cmaTypes, err := client.ContentTypes.GetCMATypes()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get cma types done")

		log.Println("get data...")
		res, err := client.Spaces.Sync("")
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get data done")

		log.Println("migrate database...")
		err = gontentful.MigratePGSQL(migrateDatabaseURL, schemaName, space, types.Items, cmaTypes.Items, res.Items, res.Token)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("migrate database done")

		log.Println("postgres schema with data successfully migrated")
	},
}
