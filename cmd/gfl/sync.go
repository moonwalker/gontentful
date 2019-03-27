package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	nextPageToken string
	execute       bool
)

type syncResponse struct {
	Sys         gontentful.Sys     `json:"sys"`
	Items       []gontentful.Entry `json:"items"`
	NextPageURL string             `json:"nextPageUrl"`
	NextSyncURL string             `json:"nextSyncUrl"`
}

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
		res, err := initSync()
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

func initSync() (*syncResponse, error) {
	init := "init"
	res, err := fetchCachedSync(init)
	if err != nil {
		return nil, err
	}
	if res == nil {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		body, err := client.Spaces.InitSync()
		if err != nil {
			return nil, err
		}

		go storeToCache(getCacheKey(init), body)

		res = &syncResponse{}
		err = json.Unmarshal(body, res)
		if err != nil {
			return nil, err
		}
	}

	if res.NextSyncURL != "" {
		go storeNextToken(res.NextSyncURL)
	}

	for res.NextPageURL != "" {
		syncToken, err := getSyncToken(res.NextPageURL)
		if err != nil {
			return nil, err
		}
		page, err := sync(syncToken)
		if err != nil {
			return nil, err
		}

		res.Items = append(res.Items, page.Items...)
		res.NextPageURL = page.NextPageURL
	}

	return res, err
}

func sync(token string) (*syncResponse, error) {
	res, err := fetchCachedSync(token)
	if err != nil {
		return nil, err
	}

	if res == nil {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		body, err := client.Spaces.Sync(token)
		if err != nil {
			return nil, err
		}

		go storeToCache(getCacheKey(token), body)

		res = &syncResponse{}
		err = json.Unmarshal(body, res)
		if err != nil {
			return nil, err
		}
	}

	for res.NextPageURL != "" {
		syncToken, err := getSyncToken(res.NextPageURL)
		if err != nil {
			return nil, err
		}
		page, err := sync(syncToken)
		if err != nil {
			return nil, err
		}

		res.Items = append(res.Items, page.Items...)
		res.NextPageURL = page.NextPageURL
		res.NextSyncURL = page.NextSyncURL
	}

	if res.NextSyncURL != "" {
		go storeNextToken(res.NextSyncURL)
	}

	return res, err
}

func getSyncToken(nextPageURL string) (string, error) {
	npu, _ := url.Parse(nextPageURL)
	m, _ := url.ParseQuery(npu.RawQuery)

	syncToken := m.Get("sync_token")
	if syncToken == "" {
		return "", fmt.Errorf("Missing sync token from response: %s", nextPageURL)
	}
	return syncToken, nil
}

func storeNextToken(nextPageURL string) {
	nextToken, err := getSyncToken(nextPageURL)
	if err == nil {
		storeToCache(getCacheKey("next_token"), []byte(nextToken))
	}
}

func fetchCachedSync(page string) (*syncResponse, error) {

	cached, err := cache.Get(getCacheKey(page))
	if err != nil {
		return nil, err
	}

	if cached != nil && len(cached) > 0 {
		res := &syncResponse{}
		err := json.Unmarshal(cached, res)
		fmt.Println("sync page ", page, " cached...")
		return res, err
	}

	return nil, nil
}

func storeToCache(key string, body []byte) {
	err := cache.Set(key, body, nil)
	if err != nil {
		fmt.Errorf("storeToCache error", err)
	}
	return
}
