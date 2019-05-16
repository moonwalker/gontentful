package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

const (
	DEFAULT_INCLUDE = 3
	MAX_INCLUDE     = 10
)

var (
	runQuery bool
	test     = []string{
		"content_type=market&fields.code=ROW&include=2&limit=1&locale=en&skip=0",
		"content_type=product&fields.name=dreamz&include=3&limit=1&skip=0",
		"content_type=game&include=3&limit=200&locale=sv&order=-fields.priority&skip=1800",
		"content_type=game&include=3&limit=200&locale=en-GB&select=sys%2Csys.id%2Cfields.slug&skip=600",
		"content_type=game&fields.content.fields.name%5Bmatch%5D=cap&fields.content.sys.contentType.sys.id=gameInfo&include=2&limit=200&locale=fi&order=-fields.priority&select=sys%2Cfields.slug%2Cfields.content%2Cfields.deviceConfigurations&skip=0",
		"content_type=game&fields.slug=winter-wonders&include=3&limit=1&locale=en&skip=0",
	}
)

func init() {
	queryCmd.AddCommand(pgQueryCmd)
}

var pgQueryCmd = &cobra.Command{
	Use:   "pg",
	Short: "Query content database",

	Run: func(cmd *cobra.Command, args []string) {
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
		if include == 0 {
			include = DEFAULT_INCLUDE
		}
		if include > MAX_INCLUDE {
			include = MAX_INCLUDE
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

		var fields []string
		fieldsQ := q.Get("select")
		q.Del("select")
		if fieldsQ != "" {
			fields = strings.Split(fieldsQ, ",")
		}

		order := q.Get("order")
		q.Del("order")

		log.Println("data gathered successfully")
		query := gontentful.NewPGQuery(schemaName, contentType, locale, defaultLocale, fields, q, order, skip, limit, include)
		log.Println("executing query schema...")
		err = query.Exec(queryDatabaseURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("query generated successfully")
	},
}
