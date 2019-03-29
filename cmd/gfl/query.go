package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	runQuery bool
	test     = []string{
		"content_type=product&query=name%3Ddreamz",
		"content_type=webComponent",
		"content_type=phrase&limit=1000&locale=en",
		"content_type=market&include=2&locale=en&query=code%3DROW",
		"content_type=page&include=5&locale=en&query=sys.id%3D5mbbjk3zNe8a6MsQS4k6Q0",
		"content_type=market",
		"content_type=locale&query=fields.disable%255Bne%255D%3Dtrue",
		"content_type=routeAlias&locale=en&query=fields.alias%3D%252F",
		"content_type=product&include=2&query=fields.name%3Ddreamz",
		"content_type=route&locale=en&query=fields.path%3D%252F",
		"content_type=locale&query=fields.code%3Den",
		"content_type=routeAlias&locale=en",
		"content_type=menu&include=2&locale=en&query=slug%3Dcasino",
		"content_type=gameStudio&limit=12&locale=en&order=-priority",
		"content_type=product&include=2&query=fields.name%3Ddreamz",
		"content_type=gameWildFeature&limit=12&locale=en&order=-priority",
		"content_type=locale&query=fields.disable%255Bne%255D%3Dtrue",
		"content_type=menu&include=2&locale=en&query=slug%3Dbottom-navigation",
		"content_type=route&locale=en&query=fields.path%3D%252F",
		"content_type=route&locale=en&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=route&locale=sv&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=route&locale=en-SE&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=route&locale=en-GB&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=route&locale=no&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=menu&include=2&locale=en&query=slug%3Dfooter-safety",
		"content_type=route&locale=fi&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=route&locale=de&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=route&locale=en-CA&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=route&locale=en-NZ&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=menu&include=2&locale=en&query=slug%3Dfooter-payment-providers",
		"content_type=route&locale=en-EU&query=id%3D426gHfnhHWWKCA2k06GIYc",
		"content_type=menu&include=2&locale=en&query=slug%3Dfooter-menu",
		"content_type=tldr&locale=en&query=key%3Dfooter-about",
		"content_type=tldr&locale=en&query=key%3Dfooter-legal",
		"content_type=menu&include=2&locale=en&query=slug%3Dfooter-social-links",
		"content_type=menu&include=2&locale=en&query=slug%3Dtop-navigation-site-desktop",
		"content_type=menu&include=2&locale=en&query=slug%3Duser-menu-mobile",
		"content_type=menu&include=2&locale=en&query=slug%3Dside-navigation-site-v2",
		"content_type=menu&include=2&locale=en&query=slug%3Dside-navigation-cashier",
		"content_type=menu&include=2&locale=en&query=slug%3Dside-navigation-account",
		"content_type=menu&include=2&locale=en&query=slug%3Dtop-navigation-user-desktop",
		"content_type=menu&include=2&locale=en&query=slug%3Dside-navigation-user",
	}
)

const (
	defaultLocale = "en"
)

func init() {
	queryCmd.PersistentFlags().BoolVarP(&runQuery, "run", "r", false, "run query")
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query content database",

	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()

		q, err := url.ParseQuery(test[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		contentType := q.Get("content_type")
		q.Del("content_type")

		locale := q.Get("locale")
		q.Del("locale")
		if locale == "" {
			locale = defaultLocale
		}

		include := 0
		includeQ := q.Get("include")
		q.Del("include")
		if len(includeQ) > 0 {
			include, _ = strconv.Atoi(includeQ)
		}

		skip := 0
		skipQ := q.Get("skip")
		q.Del("skip")
		if skipQ != "" {
			skip, _ = strconv.Atoi(skipQ)
		}

		limit := 0
		limitQ := q.Get("limit")
		q.Del("limit")
		if limitQ != "" {
			limit, _ = strconv.Atoi(limitQ)
		}

		fmt.Println(include, skip, limit)

		fields := q.Get("select")
		q.Del("select")

		order := q.Get("order")
		q.Del("order")

		metas, err := getMetadata(contentType)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		selectedFields := make([]*gontentful.PGJSONBMetaRow, 0)
		if fields != "" {
			selected := make(map[string]bool)
			for _, f := range strings.Split(fields, ",") {
				selected[f] = true
			}
			for _, m := range metas {
				if selected[m.Name] {
					selectedFields = append(selectedFields, m)
				}
			}
		} else {
			selectedFields = metas
		}

		d := time.Since(start)
		fmt.Println("data gathered successfuly in ", d.Seconds(), "s")
		split := time.Now()

		query := gontentful.NewDBContentQuery(SpaceId, contentType, locale, defaultLocale, selectedFields, q, order)

		str, err := query.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		d = time.Since(start)
		fmt.Println("query generated successfuly in ", d.Seconds(), "s")
		split = time.Now()

		if execute {
			ok, err := repo.Exec(str)
			if !ok {
				fmt.Println(err)
				os.Exit(1)
			}
			d = time.Since(split)
			fmt.Println("query executed successfuly in ", d.Seconds(), "s")
		} else {
			bytes := []byte(str)
			err := ioutil.WriteFile("/tmp/schema_query", bytes, 0644)
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

func getMetadata(contentType string) ([]*gontentful.PGJSONBMetaRow, error) {
	metaRows := make([]*gontentful.PGJSONBMetaRow, 0)
	err := repo.List(&metaRows, fmt.Sprintf(gontentful.MetaQueryFormat, SpaceId, contentType))
	if err != nil {
		return nil, err
	}
	return metaRows, nil
}
