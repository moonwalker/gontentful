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
		// "content_type=game&fields.marketCode=SE&fields.device=desktop&locale=sv&include=3&limit=0&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&include=3&limit=200&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.marketCode=ROW&fields.device=desktop&fields.slug%5Bin%5D=durian-dynamite%2Cstarburst%2Chot-nudge%2Cdeco-diamonds%2Cextra-chilli%2Cfat-rabbit%2Cbook-of-dead%2Chouse-of-doom%2Cimmortal-romance%2Csindbad&include=3&limit=200&locale=en&skip=0",
		// "content_type=gameStudio&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0&sys.id%5Bnin%5D=",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.studio.slug%5Bne%5D=netent&fields.wildFeatures.slug%5Bin%5D=walking-wilds&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live%2Cvirtual&fields.marketCode=ROW&fields.type.slug=blackjack&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live%2Cvirtual&fields.marketCode=ROW&fields.tags%5Bin%5D=low%2Ceasy&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.slug%5Bin%5D=durian-dynamite%2Cstarburst%2Chot-nudge%2Cdeco-diamonds%2Cextra-chilli%2Cfat-rabbit%2Cbook-of-dead%2Chouse-of-doom%2Cimmortal-romance%2Csindbad&include=3&limit=30&locale=en&select=sys%2Cfields.slug%2Cfields.content%2Cfields.tags%2Cfields.studio%2Cfields.deviceConfigurations&skip=0",
		// "content_type=sowQuestion&include=3&limit=200&locale=en&order=fields.id%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=routeAlias&include=3&limit=200&locale=en-GB&skip=0",
		// "content_type=route&include=3&limit=200&locale=en-GB&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=GB&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en-GB&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		//"content_type=game&fields.device=desktop&fields.marketCode=EU&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en-EU&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		//"content_type=market&include=3&limit=200&skip=0",
		//"content_type=locale&fields.disable%5Bne%5D=true&include=3&limit=200&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live&fields.marketCode=ROW&fields.type.slug%5Bin%5D=classic%2Ccraps%2Cbaccarat%2Cpoker%2Cother&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.themes.slug%5Bin%5D=world-renowned&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.wildFeatures.slug%5Bin%5D=walking-wilds&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.winFeatures.slug%5Bin%5D=grid-play&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.category.slug=slots&fields.device=desktop&fields.marketCode=ROW&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live&fields.marketCode=ROW&fields.tags%5Bin%5D=vip&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		"content_type=game&fields.device=desktop&fields.format%5Bin%5D=live%2Cvirtual&fields.marketCode=ROW&fields.type.slug=roulette&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.content.sys.contentType.sys.id=gameId&fields.deviceConfigurations.sys.id=1yyHAve4aE6AQgkIyYG4im&include=3&limit=1&select=sys%2Cfields.slug&skip=0",
		"content_type=game&fields.content.sys.contentType.sys.id=gameId&fields.deviceConfigurations.sys.id=4MYHgClCek8uqqyqAMI8ey&include=3&limit=1&select=sys%2Cfields.slug&skip=0",
		// "content_type=menu&fields.slug=casino&include=2&limit=1&locale=en-NZ&skip=0",
		"content_type=route&include=3&limit=1&locale=en-CA&skip=0&sys.id=426gHfnhHWWKCA2k06GIYc",
		"content_type=game&fields.device=desktop&fields.marketCode=ROW&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		"content_type=menuListItem&include=3&limit=200&skip=0",
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
			query := gontentful.ParsePGQuery(schemaName, defaultLocale, q)

			log.Println("executing query...")
			total, items, err := query.Exec(databaseURL)
			if err != nil {
				log.Fatal(err)
			}
			elapsed := time.Since(start)
			log.Printf("query total %d items (%d bytes) successfully in %s", total, len(items), elapsed)
		}
	},
}
