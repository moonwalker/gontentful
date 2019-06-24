package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	runQuery bool
	test     = []string{
		// "content_type=product&fields.name=dreamz&include=3&limit=1&skip=0",
		// "content_type=market&fields.code=ROW&include=2&limit=1&locale=en&skip=0",
		// "content_type=localizedSiteSetting&sys.id=35UnSWWbR6IuywWEKYKMUw&include=2&limit=1&locale=en&skip=0",
		// "content_type=game&fields.slug=winter-wonders&include=3&limit=1&locale=en&skip=0",
		// "content_type=game&include=3&limit=200&locale=sv&order=-fields.priority&skip=200",
		// "content_type=game&include=3&limit=200&locale=en-GB&select=sys%2Csys.id%2Cfields.slug&skip=600",
		// "content_type=game&fields.content.fields.name%5Bmatch%5D=jack&fields.content.sys.contentType.sys.id=gameInfo&include=2&limit=200&locale=fi&order=-fields.priority&select=sys%2Cfields.slug%2Cfields.content%2Cfields.deviceConfigurations&skip=0",
		// "content_type=localizedSiteSetting&include=2&limit=200&skip=0&sys.id=35UnSWWbR6IuywWEKYKMU",
		// "content_type=game&include=2&limit=200&skip=0&sys.id=35UnSWWbR6IuywWEKYKMU",
		// "content_type=gameWildFeature&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=routeAlias&fields.alias=%2Fcomplete-account%2F&include=3&limit=1&locale=en-SE&skip=0",
		// "content_type=userPreferences&fields.isConsent=true&include=3&limit=200&locale=en-SE&order=-fields.isConsent%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=menu&fields.slug=top-navigation-site-desktop&include=2&limit=1&locale=en-SE&skip=0",
		// "content_type=menu&fields.slug=side-navigation-user&include=2&limit=1&locale=en-SE&skip=0",
		// "content_type=menu&fields.slug=footer-menu&include=2&limit=1&locale=en-SE&skip=0",
		// "content_type=rgSetting&fields.slug=deposit-limit&include=3&limit=1&locale=en-SE&skip=0",
		// "content_type=providerIdMap&fields.slug=limitDurations&include=3&limit=1&locale=en-SE&skip=0",
		// "content_type=menu&fields.slug=footer-social-links&include=2&limit=1&locale=en-SE&skip=0",
		// "content_type=gameStudioExcludeFromMarket&fields.market=&include=3&limit=1&skip=0",
		// "content_type=routeAlias&fields.alias=%2F&include=3&limit=1&locale=en-SE&skip=0",
		"content_type=game&marketCode=SE&device=desktop&locale=sv&include=3&limit=0&skip=0",
	}
)

func init() {
	queryCmd.AddCommand(pgQueryCmd)
}

var pgQueryCmd = &cobra.Command{
	Use:   "pg",
	Short: "Query content database",

	Run: func(cmd *cobra.Command, args []string) {
		for _, q := range test {
			start := time.Now()
			log.Println("parsing", q)
			q, err := url.ParseQuery(q)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			query := gontentful.ParsePGQuery(schemaName, defaultLocale, usePreview, q)

			log.Println("executing query...")
			total, items, err := query.Exec(queryDatabaseURL)
			if err != nil {
				log.Fatal(err)
			}
			elapsed := time.Since(start)
			log.Printf("query total %d items (%d bytes) successfully in %s", total, len(items), elapsed)
		}
	},
}
