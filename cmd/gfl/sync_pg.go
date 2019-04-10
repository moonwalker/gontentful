package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

const (
	createSyncTable = "CREATE TABLE IF NOT EXISTS %s._sync ( id int primary key, token text );"
	insertSyncToken = "INSERT INTO %s._sync (id, token) VALUES (0, '%s') ON CONFLICT (id) DO UPDATE SET token = EXCLUDED.token;"
	selectSyncToken = "SELECT token FROM %s._sync WHERE id = 0;"
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

		var syncToken string
		if !initSync {
			// retrieve token from db, if exists
			if len(syncDatabaseURL) > 0 {
				log.Println("retrieving sync token...")
				db, _ := sql.Open("postgres", syncDatabaseURL)
				row := db.QueryRow(fmt.Sprintf(selectSyncToken, schemaName))
				err := row.Scan(&syncToken)
				if err != nil {
					log.Println("no sync token found")
				} else {
					log.Println("sync token found")
				}
			}
		}

		if len(syncToken) == 0 {
			log.Println("init sync...")
		} else {
			log.Println("continue sync...")
		}

		res, err := client.Spaces.Sync(syncToken)
		log.Println("sync done")
		if err != nil {
			log.Fatal(err)
		}

		log.Println("get types...")
		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get types done")

		log.Println("exec sync...")
		schema := gontentful.NewPGSyncSchema(schemaName, types.Items, res.Items)
		err = schema.Exec(syncDatabaseURL, len(syncToken) == 0)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("exec sync done")

		// save token to db, overwrite if exists
		if len(syncDatabaseURL) > 0 {
			log.Println("saving sync token...")
			db, _ := sql.Open("postgres", syncDatabaseURL)
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
