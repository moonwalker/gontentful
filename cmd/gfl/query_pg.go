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
		//"content_type=localizedSiteSetting&sys.id=35UnSWWbR6IuywWEKYKMUw&include=2&limit=1&locale=en&skip=0",
		// "content_type=game&fields.content.fields.name%5Bmatch%5D=jack&fields.content.sys.contentType.sys.id=gameInfo&include=2&limit=200&locale=fi&order=-fields.priority&select=sys%2Cfields.slug%2Cfields.content%2Cfields.deviceConfigurations&skip=0",
		"content_type=market&fields.code=ROW&include=2&limit=1&locale=en&skip=0",
		// "content_type=product&fields.name=dreamz&include=3&limit=1&skip=0",
		// "content_type=game&include=3&limit=200&locale=sv&order=-fields.priority&skip=200",
		// "content_type=game&include=3&limit=200&locale=en-GB&select=sys%2Csys.id%2Cfields.slug&skip=600",
		// "content_type=game&fields.slug=winter-wonders&include=3&limit=1&locale=en&skip=0",
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
			items, total, err := query.Exec(queryDatabaseURL)
			if err != nil {
				log.Fatal(err)
			}
			elapsed := time.Since(start)
			log.Printf("query returned %d items (%d total) successfully in %s", len(items), total, elapsed)
		}
	},
}

// DROP TYPE IF EXISTS _meta CASCADE

// CREATE TYPE _meta AS (
// 	name TEXT,
// 	link_type TEXT,
// 	items TEXT,
// 	is_localized BOOLEAN);

// DROP FUNCTION content.__test_get_metas(text)

// CREATE OR REPLACE FUNCTION content.__test_get_metas(tableName text)
// RETURNS SETOF _meta AS $$
// BEGIN
// 	 RETURN QUERY EXECUTE 'SELECT
// 		name,
// 		link_type,
// 		items ->> ''linkType'' as items,
// 		is_localized
//         FROM content.' || tableName || '___meta';

// END;
// $$ LANGUAGE 'plpgsql';

// SELECT * FROM content.__test_get_metas('product');

// CREATE OR REPLACE FUNCTION content.__test_meta(tableName text, locale text, defaultLocale text, usePreview boolean)
// RETURNS text AS $$
// DECLARE
// 	qs text;
// 	suffix text := '__publish';
// 	isFirst boolean := true;
// 	meta _meta;
// BEGIN
// 	IF usePreview THEN
// 		suffix := '';
// 	END IF;

// 	qs := 'SELECT ';
// 	--FOREACH meta SLICE 1 IN ARRAY metas LOOP
// 	FOR meta IN SELECT * FROM content.__test_get_metas(tableName) LOOP
// 	    IF isFirst THEN
// 	    	isFirst := false;
// 	    ELSE
// 	    	qs := qs || ', ';
// 	    END IF;

// 		IF meta.is_localized AND locale <> defaultLocale THEN
// 			qs := 'COALESCE(' || tableName || '_' || locale || '.' || meta.name || ',' ||
// 			tableName || '_' || defaultLocale || '.' || meta.name || ')';
// 		ELSE
// 	    	qs := qs || tableName || '_' || defaultLocale || '.' || meta.name;
// 		END IF;

// 		qs := qs || ' as ' || meta.name;
//     END LOOP;

// 	qs := qs || ' FROM content.' || tableName || '_' || defaultLocale || suffix || ' ' || tableName || '_' || defaultLocale;

// 	IF locale <> defaultLocale THEN
// 		qs := qs || ' LEFT JOIN content.' || tableName || '_' || locale || '__publish ' || tableName || '_' || locale ||
// 		' ON ' || tableName || '_' || defaultLocale || '.sys_id = ' || tableName || '_' || locale || '.sys_id';
// 	END IF;

// 	RETURN qs;
// END;
// $$ LANGUAGE 'plpgsql';

// CREATE OR REPLACE FUNCTION content.__test_locale(locale text, defaultLocale text, filters text[])
// RETURNS SETOF content.locale_en__publish AS $$
// DECLARE
// 	qs text;
// BEGIN

// 	RETURN qs;
// END;
// $$ LANGUAGE 'plpgsql';

// CREATE OR REPLACE FUNCTION content.__test_market(locale text, defaultLocale text, filters text[], )
// --RETURNS SETOF RECORD AS $$
// RETURNS SETOF content.market_en__publish AS $$
// DECLARE
// 	qs text;
// 	firstFilter boolean := true;
// 	filter text;
// 	js text[];
// BEGIN

// 	qs := content.__test_meta(tableName, locale, defaultLocale, false);

// 	IF joins IS NOT NULL THEN
// 		FOREACH js SLICE 1 IN ARRAY joins LOOP
// 			qs := qs || ' LEFT JOIN content.' || js[2] || '_' || defaultLocale || '__publish ' || js[2] || '_' || defaultLocale  || ' ON ' || tableName || '_' || defaultLocale || '.' || js[1] || ' = ' || js[2] || '_' || defaultLocale || '.sys_id';
// 			IF locale <> defaultLocale THEN
// 				qs := qs || ' LEFT JOIN content.' || js[2] || '_' || locale || '__publish ' || js[2] || '_' || locale  || ' ON ' || tableName || '_' || locale || '.' || js[1] || ' = ' || js[2] || '_' || locale || '.sys_id';
// 			END IF;
// 		END LOOP;
// 	END IF;

// 	RAISE NOTICE '%', qs;

// 	IF filters IS NOT NULL THEN
// 		qs := qs || ' WHERE ';
// 		FOREACH filter IN ARRAY filters LOOP
// 			IF firstFilter THEN
// 				firstFilter:= false;
// 			ELSE
// 				qs := qs || ' AND ';
// 			END IF;
// 			qs := qs || filter;
// 		END LOOP;
// 	END IF;

// 	RAISE NOTICE '%', qs;

// 	RETURN QUERY EXECUTE qs;
// END;
// $$ LANGUAGE 'plpgsql';

// CREATE OR REPLACE FUNCTION content.__test_product(locale text, defaultLocale text, filters text[], include INTEGER, level INTEGER)
// RETURNS TABLE(
// 	name TEXT,
// 	type TEXT,
// 	markets JSON[],
// 	default_locale JSON,
// 	market_blacklist JSON[],
// 	base_currency JSON,
// 	enforce_market BOOLEAN,
// 	show_market_selector BOOLEAN,
// 	api_keys JSON
// )AS $$
// DECLARE
// 	qs text;
// BEGIN
// 	--CREATE TEMP TABLE _market_meta_holder ON COMMIT DROP AS
// 	--SELECT * FROM content.__test_get_metas('product');

// 	qs := content.__test_meta('product', locale, defaultLocale, false);

// 	RAISE NOTICE '%', qs;

// 	RETURN QUERY EXECUTE qs;
// END;
// $$ LANGUAGE 'plpgsql';

// SELECT ROW(name, );
// SELECT * FROM content.__test_product('sv', 'en', ARRAY['name = ''dreamz'''], 3, 0);
// SELECT * FROM content.__test_meta('product', 'sv', 'en', false);

// SELECT array_agg(json_build_object(
//     'name', metas.name,
//     'link_type', metas.link_type,
//     'items', metas.items,
//     'is_localized', metas.is_localized
// )) FROM content.__test_get_metas('product') metas;

// SELECT m from (SELECT CAST(ROW(name,link_type,items, is_localized) AS _meta) from content.__test_get_metas('product')) AS m;

// SELECT
// market_en.name as name,
// market_en.code as code,
// array_agg(json_build_object(
//     'code', locale_en.code,
//     'code_alias', locale_en.code_alias,
//     'name', locale_en.name,
//     'exclude_from_sitemap', locale_en.exclude_from_sitemap,
//     'canonical_locale', locale_en.canonical_locale,
// 	'disable', locale_en.disable,
// 	'language_id', locale_en.language_id
// )) AS locales,
// json_build_object(
//     'name', localized_site_setting_en.name,
// 'domain',localized_site_setting_en.domain,
// 'title',localized_site_setting_en.title,
// 'description',localized_site_setting_en.description,
// 'background_image',localized_site_setting_en.background_image,
// 'logo',localized_site_setting_en.logo,
// 'image',localized_site_setting_en.image
// ) AS localized_site_setting

// FROM content.market_en__publish market_en
// LEFT JOIN content.locale_en__publish locale_en ON locale_en.sys_id = ANY(market_en.locales)
// LEFT JOIN content.localized_site_setting_en__publish localized_site_setting_en ON localized_site_setting_en.sys_id = market_en.localized_site_setting
// WHERE (market_en.code = 'ROW')
// GROUP BY market_en.name, market_en.code, localized_site_setting_en.name, localized_site_setting_en.domain, localized_site_setting_en.title, localized_site_setting_en.description, localized_site_setting_en.background_image, localized_site_setting_en.logo, localized_site_setting_en.image, is_default, auth_provider_enforced, payment_settings, market_en.bank_id_available;
