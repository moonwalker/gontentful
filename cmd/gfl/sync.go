package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	syncNextPageCmd.PersistentFlags().StringVarP(&nextPageToken, "page", "p", "", "next page token")
	syncCmd.AddCommand(syncNextPageCmd)
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
		start := time.Now()
		fmt.Println("starting initSync...")
		res, err := sync("")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		d := time.Since(start)
		fmt.Println("initSync executed successfuly in ", d.Seconds(), "s")
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

		fmt.Println("creating schema...")

		schema := gontentful.NewPGSyncSchema(schemaName, assetTableName, space, res.Items)

		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		d = time.Since(split)
		fmt.Println("schema rendered successfuly in ", d.Seconds(), "s")
		split = time.Now()

		bytes := []byte(str)
		err = ioutil.WriteFile("/tmp/schema_initsync", bytes, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// fmt.Println(str)

		ok, err := repo.Exec(str)
		if !ok {
			fmt.Println(err)
			os.Exit(1)
		}
		d = time.Since(split)
		fmt.Println("script executed successfuly in ", d.Seconds(), "s")

		d = time.Since(start)
		fmt.Println("completed successfuly in ", d.Seconds(), "s")
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

		d := time.Since(start)
		fmt.Println("sync token executed successfuly in ", d.Seconds(), "s")
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

		schema := gontentful.NewPGSyncSchema(schemaName, assetTableName, space, res.Items)

		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		d = time.Since(split)
		fmt.Println("schema rendered successfuly in ", d.Seconds(), "s")
		split = time.Now()

		bytes := []byte(str)
		err = ioutil.WriteFile("/tmp/schema_sync", bytes, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		// fmt.Println(str)

		ok, err := repo.Exec(str)
		if !ok {
			fmt.Println(err)
			os.Exit(1)
		}
		d = time.Since(split)
		fmt.Println("script executed successfuly in ", d.Seconds(), "s")

		d = time.Since(start)
		fmt.Println("completed successfuly in ", d.Seconds(), "s")
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
