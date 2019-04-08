package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	execSchema bool
)

func init() {
	schemaCmd.AddCommand(pgSchemaCmd)
	schemaCmd.AddCommand(gqlSchemaCmd)
	schemaCmd.AddCommand(jsonbSchemaCmd)
	schemaCmd.PersistentFlags().BoolVarP(&execSchema, "exec", "e", false, "execute schema")
	schemaCmd.PersistentFlags().StringVarP(&databaseURL, "url", "u", defaultPostgresURL, "database url")
	rootCmd.AddCommand(schemaCmd)
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Creates schema from contentful types",
}

var pgSchemaCmd = &cobra.Command{
	Use:   "pg",
	Short: "Creates postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		if execSchema {
			log.Println("creating postgres schema...")
		}

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		space, err := client.Spaces.GetSpace()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		types, err := client.ContentTypes.GetTypes()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schema := gontentful.NewPGSQLSchema(schemaName, assetTableName, space, types.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if !execSchema {
			fmt.Println(str)
			return
		} else {
			log.Println("postgres schema successfully created")
		}

		log.Println("executing postgres schema...")
		db, _ := sql.Open("postgres", databaseURL)
		txn, err := db.Begin()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		_, err = db.Exec(str)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = txn.Commit()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		log.Println("postgres schema successfully executed")
	},
}

var gqlSchemaCmd = &cobra.Command{
	Use:   "gql",
	Short: "Creates graphql schema",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		data, err := client.ContentTypes.Get(nil)
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

		schema := gontentful.NewGraphQLSchema(resp.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(str)
	},
}

var jsonbSchemaCmd = &cobra.Command{
	Use:   "jsonb",
	Short: "Creates postgres jsonb schema",

	Run: func(cmd *cobra.Command, args []string) {

		resp, err := fetchCachedContentTypes()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if resp == nil {
			resp, err = fetchContentTypes()
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		schema := gontentful.NewPGJSONBSchema(schemaName, assetTableName, resp.Items)
		str, err := schema.Render()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if execSchema {
			// repo, err := dal.NewPostgresRepo()
			// if err != nil {
			// 	fmt.Println(err)
			// 	os.Exit(1)
			// }

			// ok, err := repo.Exec(str)
			// if !ok {
			// 	fmt.Println(err)
			// 	os.Exit(1)
			// }
			fmt.Println("schema created succesfuly")
		} else {
			bytes := []byte(str)
			err := ioutil.WriteFile("/tmp/schema_jsonb", bytes, 0644)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(str)
		}
	},
}

func fetchContentTypes() (*gontentful.ContentTypes, error) {
	client := gontentful.NewClient(&gontentful.ClientOptions{
		CdnURL:   apiURL,
		SpaceID:  SpaceId,
		CdnToken: CdnToken,
	})

	data, err := client.ContentTypes.Get(nil)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	resp := &gontentful.ContentTypes{}
	err = json.Unmarshal(data, resp)

	go cache.Set(fmt.Sprintf("sync_%s:content_types", SpaceId), data, nil)

	return resp, err
}

func fetchCachedContentTypes() (*gontentful.ContentTypes, error) {
	key := fmt.Sprintf("sync_%s:content_types", SpaceId)

	cached, err := cache.Get(key)
	if err != nil {
		return nil, err
	}
	res := &gontentful.ContentTypes{}
	if cached != nil {
		err := json.Unmarshal(cached, res)
		fmt.Println("content types cached...")
		return res, err
	}

	return nil, nil
}
