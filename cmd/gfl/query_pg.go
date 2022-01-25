package main

import (
	"database/sql"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	runQuery bool
	test     = []string{
		"content_type=valuableFreespin&fields.environment%5Bne%5D=staging&fields.excludeFromSegmentationAction%5Bne%5D=yes&fields.value%5Blte%5D=1.000000&include=1&limit=1000&locale=en&order=-fields.value%2C-sys.updatedAt%2C-sys.createdAt%2Csys.id&skip=0",
		// "content_type=gameWildFeature&fields.name%5Bmatch%5D=%27fruits&include=1&limit=1000&locale=en-EU&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=gameStudio&fields.enabled%5Bne%5D=false&fields.name%5Bmatch%5D=dragon%27s+&include=1&limit=1000&locale=en-EU&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0&sys.id%5Bnin%5D=2JsfZhGIsM84gk8MAWy6UY%2C1n7p035oIQ46kKUSKg4Csa%2C2szkSRoCiY0MwkegOKe6im%2C6QsVoUKraoOIq04S6yGYKM%2CLcaJjLfZ82IawQ0MuUqCc%2C4XKBCEZTM3cHSoFh3v1doI%2C2YO6k4bCC54QkbhIx8Ncqg%2C3X24TeiCsrMhIHgYITHOlc%2C3YtvKTXi08myuwKY8YUI4S%2C3K6a1VGwdymQeCamyOSMAs%2C2KID4jvLGUEOuCSI6YEuKo%2C5eDdC3owGci4WwkiUOAYcm%2CE1RRDLNADQiqsmCGoM2am%2Cm2M8Q8JDWWTLC9TPtWAwn%2C3xqNdiQ7CEE8Gy6A4wgKeo%2CdsGjjzpDQPIELDdxbiQbx%2C6YVKxuzJ3ixJPJUJ3LCA98",
		// "content_type=valuableDepositBonus&fields.slug%5Bin%5D=100p_20_250_40x&include=3&limit=1000&locale=en&skip=0",
		// "content_type=product&fields.name=dreamz&include=3&limit=1&skip=0",
		// "content_type=webComponent&include=3&limit=1000&skip=0",
		// "content_type=phrase&include=3&limit=1000&locale=en&skip=0",
		// "content_type=route&include=3&limit=1000&locale=en&skip=0",
		// "content_type=market&fields.code=ROW&include=2&limit=1&locale=en&skip=0",
		// "content_type=currency&include=3&limit=1000&locale=en&skip=0",
		// "content_type=market&include=3&limit=1000&skip=0",
		// "content_type=locale&fields.disable%5Bne%5D=true&include=3&limit=1000&skip=0",
		// "content_type=page&include=5&limit=1&locale=en&skip=0&sys.id=5mbbjk3zNe8a6MsQS4k6Q0",
		// "content_type=locale&fields.code=en&include=3&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=user-menu-mobile-v2&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=top-navigation-site-desktop&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=top-navigation-user-desktop&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=casino&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=footer-menu&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=footer-safety&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=footer-payment-providers&include=2&limit=1&locale=en&skip=0",
		// "content_type=tldr&fields.key=footer-legal&include=3&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=footer-social-links&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=account-menu-bottom-bar&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=side-navigation-user&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=side-navigation-site-v2&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=side-navigation-account&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=side-navigation-cashier&include=2&limit=1&locale=en&skip=0",
		// "content_type=gameStudioExcludeFromMarket&fields.market=ROW&include=3&limit=1&skip=0",
		// "content_type=gameStudio&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0&sys.id%5Bnin%5D=2YqnioGjW8wMoWQU4gqI8C%2C1nvcGNYTL2ygaQ8MIcKMUY",
		// "content_type=gameWildFeature&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=gameWinFeature&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=gameBonusFeature&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=gameStudioExcludeFromMarket&fields.market=CA&include=3&limit=1&skip=0",
		// "content_type=gameStudio&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0&sys.id%5Bnin%5D=1nvcGNYTL2ygaQ8MIcKMUY%2C2YqnioGjW8wMoWQU4gqI8C",
		// "content_type=gameTheme&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=page&include=5&limit=1&locale=en&skip=0&sys.id=4RYB7CHpeM6cMckQyOy20s",
		// "content_type=menu&fields.slug=account&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=bottom-navigation&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=document-upload&include=2&limit=1&locale=en&skip=0",
		// "content_type=menu&fields.slug=cashier&include=2&limit=1&locale=en&skip=0",
		// "content_type=page&include=5&limit=1&locale=en&skip=0&sys.id=3PHIIsENlmCO84A8gKkkWw",
		// "content_type=valuableFreespin&fields.slug%5Bin%5D=1_cash-vandal_0.1x10_0w%2C1_starburst_0.1x10_0w%2C2.5_skulls-up_0.1x25_0w%2C5_zeus-lightning-power-reels_1x5_0w%2C2.5_dawn-of-egypt_0.1x25_0w&include=3&limit=1000&locale=en&skip=0",
		// "content_type=page&include=5&limit=1&locale=en&skip=0&sys.id=2NmoamlIkgwUgyEYYcmI2q",
		// "content_type=menu&fields.slug=valuables&include=2&limit=1&locale=en&skip=0",
		// "content_type=valuableFreespin&fields.slug%5Bin%5D=1_cash-vandal_0.1x10_0w%2C1_starburst_0.1x10_0w%2C2.5_skulls-up_0.1x25_0w%2C5_zeus-lightning-power-reels_1x5_0w%2C2.5_dawn-of-egypt_0.1x25_0w%2C1_wild-frames_0.2x5_0w%2C1_gonzos-quest-megaways_0.1x10_0w%2C5_esqueleto-explosivo-2_0.1x50_0w&include=3&limit=1000&locale=en&skip=0",
		// "content_type=gameStudio&fields.slug=netent&include=3&limit=1000&locale=en&skip=0",
		// "content_type=gameBonusFeature&fields.slug=feature-drop&include=3&limit=1000&locale=en&skip=0",
		// "content_type=gameWinFeature&fields.slug=sticky-symbols&include=3&limit=1000&locale=en&skip=0",
		// "content_type=gameWildFeature&fields.slug=special-wilds&include=3&limit=1000&locale=en&skip=0",
		// "content_type=gameTheme&fields.slug=dreams&include=3&limit=1000&locale=en&skip=0",
		// "content_type=sowQuestion&include=3&limit=200&locale=en&order=fields.id%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=userPreferences&fields.isConsent=true&include=3&limit=200&locale=en-SE&order=-fields.isConsent%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&include=2&limit=200&skip=0&sys.id=3FxOOrQG3K60Coe6eU8oCM",
		// "content_type=game&fields.slug=winter-wonders&include=3&limit=1&locale=en&skip=0",

		// "content_type=game&fields.marketCode=SE&fields.device=desktop&locale=sv&include=3&limit=0&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&include=3&limit=200&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.marketCode=ROW&fields.device=desktop&fields.slug%5Bin%5D=durian-dynamite%2Cstarburst%2Chot-nudge%2Cdeco-diamonds%2Cextra-chilli%2Cfat-rabbit%2Cbook-of-dead%2Chouse-of-doom%2Cimmortal-romance%2Csindbad&include=3&limit=200&locale=en&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.studio.slug%5Bne%5D=netent&fields.wildFeatures.slug%5Bin%5D=walking-wilds&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live%2Cvirtual&fields.marketCode=ROW&fields.type.slug=blackjack&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live%2Cvirtual&fields.marketCode=ROW&fields.tags%5Bin%5D=low%2Ceasy&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.slug%5Bin%5D=durian-dynamite%2Cstarburst%2Chot-nudge%2Cdeco-diamonds%2Cextra-chilli%2Cfat-rabbit%2Cbook-of-dead%2Chouse-of-doom%2Cimmortal-romance%2Csindbad&include=3&limit=30&locale=en&select=sys%2Cfields.slug%2Cfields.content%2Cfields.tags%2Cfields.studio%2Cfields.deviceConfigurations&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=GB&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en-GB&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=EU&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en-EU&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live&fields.marketCode=ROW&fields.type.slug%5Bin%5D=classic%2Ccraps%2Cbaccarat%2Cpoker%2Cother&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.themes.slug%5Bin%5D=world-renowned&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.wildFeatures.slug%5Bin%5D=walking-wilds&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.winFeatures.slug%5Bin%5D=grid-play&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.category.slug=slots&fields.device=desktop&fields.marketCode=ROW&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&fields.tags%5Bin%5D=top&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live&fields.marketCode=ROW&fields.tags%5Bin%5D=vip&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.device=desktop&fields.format%5Bin%5D=live%2Cvirtual&fields.marketCode=ROW&fields.type.slug=roulette&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
		// "content_type=game&fields.content.sys.contentType.sys.id=gameId&fields.deviceConfigurations.sys.id=1yyHAve4aE6AQgkIyYG4im&include=3&limit=1&select=sys%2Cfields.slug&skip=0",
		// "content_type=game&fields.content.sys.contentType.sys.id=gameId&fields.deviceConfigurations.sys.id=4MYHgClCek8uqqyqAMI8ey&include=3&limit=1&select=sys%2Cfields.slug&skip=0",
		// "content_type=game&include=3&limit=200&locale=sv&order=-fields.priority&skip=200",
		// "content_type=game&include=3&limit=200&locale=en-GB&select=sys%2Csys.id%2Cfields.slug&skip=600",
		// "content_type=game&fields.content.fields.name%5Bmatch%5D=jack&fields.content.sys.contentType.sys.id=gameInfo&include=2&limit=200&locale=fi&order=-fields.priority&select=sys%2Cfields.slug%2Cfields.content%2Cfields.deviceConfigurations&skip=0",
		// "content_type=game&fields.device=desktop&fields.marketCode=ROW&include=3&limit=12&locale=en&order=-fields.priority%2C-sys.updatedAt%2Csys.id&skip=0",
	}
)

func init() {
	queryCmd.AddCommand(pgQueryCmd)
}

func execQuery(q string) int64 {
	start := time.Now()
	// log.Println("parsing", q)
	qv, err := url.ParseQuery(q)
	if err != nil {
		log.Fatal(err)
	}
	query := gontentful.ParsePGQuery(schemaName, "en", qv)
	// log.Println("executing query...")
	_, _, err = query.Exec(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(start).Milliseconds()
	// log.Printf("query %s total %d items (%d bytes) successfully in %dms", q, total, len(items), elapsed)
	return elapsed
}

func execQueryFake(q string) int64 {
	start := time.Now()
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var _id string
	res := db.QueryRow("SELECT _id from content.currency where _id = '32BGobyvxSA2WgCsQK0Iso_en'")
	err = res.Scan(&_id)
	if err != nil && err != sql.ErrNoRows {
		log.Fatal(err)
	}
	elapsed := time.Since(start).Milliseconds()
	return elapsed
}

var pgQueryCmd = &cobra.Command{
	Use:   "pg",
	Short: "Query content database",

	Run: func(cmd *cobra.Command, args []string) {
		runTimes := make(map[string][]int64)
		l := len(test)
		u := 1
		for i := 0; i < u; i++ {
			var wg sync.WaitGroup
			wg.Add(l)
			log.Printf("executing %d queries async...", l)
			for _, qs := range test {
				if runTimes[qs] == nil {
					runTimes[qs] = []int64{
						0,
						0,
					}
				}
				go func(q string) {
					defer wg.Done()
					runTimes[q][0] += execQuery(q)
				}(qs)
			}
			wg.Wait()
			log.Printf("executing %d queries sync...", l)
			for _, q := range test {
				if runTimes[q] == nil {
					runTimes[q] = []int64{
						0,
						0,
					}
				}
				runTimes[q][1] += execQuery(q)
			}
		}
		ui := int64(u)
		for k, v := range runTimes {
			log.Printf("query %s async %d vs sync %d diff %d", k, v[0]/ui, v[1]/ui, (v[0]-v[1])/ui)
		}
	},
}
