package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	initSync, cacheResp bool
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
			CdnURL:        apiURL,
			SpaceID:       spaceID,
			EnvironmentID: environmentID,
			CdnToken:      cdnToken,
		})

		var err error
		var syncToken string
		if !initSync {
			// retrieve token from db, if exists
			if len(databaseURL) > 0 {
				log.Println("retrieving sync token...")
				syncToken, err = gontentful.GetSyncToken(databaseURL, schemaName)
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
		if err != nil {
			log.Fatal(err)
		}
		log.Println("sync done")

		// for _, i := range res.Items {
		// 	fmt.Println(i.Sys.ID)
		// }

		log.Println("get space...")
		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("get types...")
		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get types done")

		log.Println("exec...")
		schema := gontentful.NewPGSyncSchema(schemaName, space.Locales, types.Items, res.Items, len(syncToken) == 0)
		err = schema.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("exec done")

		// save token to db, overwrite if exists
		if len(databaseURL) > 0 {
			log.Println("saving sync token...")
			err = gontentful.SaveSyncToken(databaseURL, schemaName, res.Token)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("sync token saved")
		}
	},
}
