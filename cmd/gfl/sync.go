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
	execute       bool
)

func getCacheKey(page string) string {
	return fmt.Sprintf("sync_%s:%s", SpaceId, page)
}

func init() {
	initSyncCmd.PersistentFlags().BoolVarP(&execute, "exec", "e", false, "execute script")
	syncCmd.AddCommand(initSyncCmd)
	syncNextPageCmd.PersistentFlags().StringVarP(&nextPageToken, "page", "p", "", "next page token")
	syncNextPageCmd.PersistentFlags().BoolVarP(&execute, "exec", "e", false, "execute script")
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

		schema := gontentful.NewPGSyncSchema(SpaceId, assetTableName, res.Items)

		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		d = time.Since(split)
		fmt.Println("schema rendered successfuly in ", d.Seconds(), "s")
		split = time.Now()
		if execute {
			ok, err := repo.Exec(str)
			if !ok {
				fmt.Println(err)
				os.Exit(1)
			}
			d = time.Since(split)
			fmt.Println("script executed successfuly in ", d.Seconds(), "s")
		} else {
			bytes := []byte(str)
			err := ioutil.WriteFile("/tmp/schema_initsync", bytes, 0644)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(str)
		}
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

		schema := gontentful.NewPGSyncSchema(SpaceId, assetTableName, res.Items)

		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		d = time.Since(split)
		fmt.Println("schema rendered successfuly in ", d.Seconds(), "s")
		split = time.Now()

		if execute {
			ok, err := repo.Exec(str)
			if !ok {
				fmt.Println(err)
				os.Exit(1)
			}
			d = time.Since(split)
			fmt.Println("script executed successfuly in ", d.Seconds(), "s")
		} else {
			bytes := []byte(str)
			err := ioutil.WriteFile("/tmp/schema_sync", bytes, 0644)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(str)
		}
		d = time.Since(start)
		fmt.Println("completed successfuly in ", d.Seconds(), "s")
	},
}

func sync(token string) (*gontentful.SyncResponse, error) {
	key := token
	if key == "" {
		key = "initial"
	}
	client := gontentful.NewClient(&gontentful.ClientOptions{
		CdnURL:   apiURL,
		SpaceID:  SpaceId,
		CdnToken: CdnToken,
	})
	res, err := fetchCachedSync(key)
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = &gontentful.SyncResponse{}
		nextSyncToken, err := client.Sync.Sync(token, syncCallback(res, key))
		if err != nil {
			return nil, err
		}
		if nextSyncToken != "" {
			go storeToCache(getCacheKey("next_token"), []byte(nextSyncToken))
		}
	}

	return res, err
}

func syncCallback(res *gontentful.SyncResponse, initKey string) func(*gontentful.SyncResponse) error {
	key := initKey
	return func(syncRes *gontentful.SyncResponse) error {
		if syncRes.NextPageURL != "" {
			go storeSyncResponse(getCacheKey(key), syncRes)
			key = syncRes.NextPageURL
		}
		res.Items = append(res.Items, syncRes.Items...)
		return nil
	}
}

func fetchCachedSync(key string) (*gontentful.SyncResponse, error) {

	cached, err := cache.Get(getCacheKey(key))
	if err != nil {
		return nil, err
	}

	if cached != nil && len(cached) > 0 {
		res := &gontentful.SyncResponse{}
		err := json.Unmarshal(cached, res)
		for res.NextPageURL != "" {
			body, err := cache.Get(getCacheKey(res.NextPageURL))
			if err != nil {
				return nil, err
			}
			page := &gontentful.SyncResponse{}
			err = json.Unmarshal(body, err)
			if err != nil {
				return nil, err
			}
			res.NextPageURL = page.NextPageURL
			res.NextSyncURL = page.NextSyncURL
			res.Items = append(res.Items, page.Items...)
		}
		return res, err
	}

	return nil, nil
}

func storeSyncResponse(key string, res *gontentful.SyncResponse) {
	body, err := json.Marshal(res)
	if err != nil {
		fmt.Errorf("Marshal error", err)
		return
	}
	storeToCache(key, body)
	return
}

func storeToCache(key string, body []byte) {
	err := cache.Set(key, body, nil)
	if err != nil {
		fmt.Errorf("storeToCache error", err)
	}
	return
}
