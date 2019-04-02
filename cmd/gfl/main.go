package main

import (
	"fmt"
	"os"

	"github.com/moonwalker/backend/pkg/dal"

	"github.com/moonwalker/backend/pkg/store"
	"github.com/moonwalker/backend/pkg/store/redis"
)

var (
	SpaceId  string
	CdnToken string
	cache    store.Store
	repo     dal.Repository
)

const (
	apiURL         = "cdn.contentful.com"
	assetTableName = "_assets"
	schemaName     = "content"
)

func init() {
	cache = getCache()
	r, err := dal.NewPostgresRepo()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	repo = r
	rootCmd.PersistentFlags().StringVarP(&SpaceId, "space", "s", "", "cf space id (required)")
	rootCmd.PersistentFlags().StringVarP(&CdnToken, "token", "t", "", "token token (required)")
	rootCmd.MarkFlagRequired("space")
	rootCmd.MarkFlagRequired("token")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getCache() store.Store {
	redisURL := os.Getenv("CACHE_URL")
	if len(redisURL) == 0 {
		redisURL = "redis://127.0.0.1:6379"
	}
	return redistore.New(redisURL)
}
