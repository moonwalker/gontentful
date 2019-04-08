package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	nextPageToken string
)

func getCacheKey(page string) string {
	return fmt.Sprintf("sync_%s:%s", SpaceId, page)
}

func init() {
	syncCmd.AddCommand(initSyncCmd)
	syncCmd.AddCommand(syncNextPageCmd)
	syncCmd.PersistentFlags().StringVarP(&databaseURL, "url", "u", defaultPostgresURL, "database url")
	syncNextPageCmd.PersistentFlags().StringVarP(&nextPageToken, "page", "p", "", "next page token")
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync contentful data",
}

var initSyncCmd = &cobra.Command{
	Use:   "init",
	Short: "Run initial sync",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		log.Println("get space...")
		space, err := client.Spaces.GetSpace()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		log.Println("get space done")

		log.Println("get types...")
		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		log.Println("get types done")

		log.Println("init sync...")
		res, err := client.Spaces.Sync("")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		log.Println("init sync done")

		log.Println("bulk insert...")
		schema := gontentful.NewPGSyncSchema(schemaName, assetTableName, space, types.Items, res.Items)
		err = schema.BulkInsert(databaseURL)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		log.Println("bulk insert done")
	},
}

var syncNextPageCmd = &cobra.Command{
	Use:   "next",
	Short: "Sync next page",
	Long:  "Optional next page token can be provided. If not will try to use `next_sync_token` from cache.",

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("starting sync with token: ", nextPageToken)
		start := time.Now()
		token := nextPageToken
		if token == "" {
			body, err := cache.Get("next_sync_token")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if body != nil && len(body) > 0 {
				token = string(body)
			}
			if token == "" {
				fmt.Println("missing token. page not provided and next_sync_token cannot be found in cache")
				os.Exit(1)
			}
		}

		res, err := sync(token)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(res)

		d := time.Since(start)
		fmt.Println("sync token executed successfully in ", d.Seconds(), "s")
		split := time.Now()

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		data, err := client.Spaces.Get(nil)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		space := &gontentful.Space{}
		err = json.Unmarshal(data, space)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		data, err = client.ContentTypes.Get(nil)
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

		// schema := gontentful.NewPGSyncSchema(schemaName, assetTableName, space, res.Items)

		// fmt.Println(str)
		fmt.Println("executing schema...")
		// repo, err := dal.NewPostgresRepo()
		// if err != nil {
		// 	fmt.Println(err)
		// 	os.Exit(1)
		// }
		// ok, err := repo.Exec(str)
		// if !ok {
		// 	fmt.Println(err)
		// 	os.Exit(1)
		// }
		d = time.Since(split)
		fmt.Println("script executed successfully in ", d.Seconds(), "s")

		d = time.Since(start)
		fmt.Println("completed successfully in ", d.Seconds(), "s")
	},
}

func sync(token string) (*gontentful.SyncResult, error) {
	key := token
	if key == "" {
		key = "initial"
	}
	cacheKey := getCacheKey(key)
	client := gontentful.NewClient(&gontentful.ClientOptions{
		CdnURL:   apiURL,
		SpaceID:  SpaceId,
		CdnToken: CdnToken,
	})
	res, err := fetchCachedSync(cacheKey)
	if err != nil {
		return nil, err
	}
	if res == nil {
		res, err = client.Spaces.Sync(token)
		if err != nil {
			return nil, err
		}

		storeSyncResponse(cacheKey, res)
		if res.Token != "" {
			storeToCache(getCacheKey("next_token"), []byte(res.Token))
		}
	}
	return res, err
}

func fetchCachedSync(key string) (*gontentful.SyncResult, error) {
	cached, err := cache.Get(key)
	if err != nil {
		return nil, err
	}
	if cached != nil && len(cached) > 0 {
		res := &gontentful.SyncResult{}
		first := &gontentful.SyncResponse{}
		err := json.Unmarshal(cached, first)
		nextPageURL := first.NextPageURL
		res.Items = append(res.Items, first.Items...)
		for nextPageURL != "" {
			body, err := cache.Get(getCacheKey(nextPageURL))
			if err != nil {
				return nil, err
			}
			page := &gontentful.SyncResponse{}
			err = json.Unmarshal(body, err)
			if err != nil {
				return nil, err
			}
			nextPageURL = page.NextPageURL
			res.Items = append(res.Items, page.Items...)
		}
		return res, err
	}
	return nil, nil
}

func storeSyncResponse(key string, res *gontentful.SyncResult) {
	body, err := json.Marshal(res)
	if err != nil {
		fmt.Println(fmt.Errorf("Marshal error: %s", err))
		return
	}
	storeToCache(key, body)
	return
}

func storeToCache(key string, body []byte) {
	err := cache.Set(key, body, nil)
	if err != nil {
		fmt.Println(fmt.Errorf("storeToCache error: %s", err))
	}
	return
}
