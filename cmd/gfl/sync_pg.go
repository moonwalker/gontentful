package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

const (
	createSyncTable = "CREATE TABLE IF NOT EXISTS %s._sync ( token text );"
	insertSyncToken = "INSERT INTO %s._sync (token) VALUES (%s);"
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

		// store token to db
		if len(databaseURL) > 0 {
			log.Println("saving sync token...")
			db, _ := sql.Open("postgres", databaseURL)
			_, err := db.Exec(fmt.Sprintf(createSyncTable, schemaName))
			if err != nil {
				log.Fatal(err)
			}
			_, err = db.Exec(fmt.Sprintf(insertSyncToken, schemaName, res.Token))
			if err != nil {
				log.Fatal(err)
			}
			log.Println("sync token saved")
		}
	},
}
