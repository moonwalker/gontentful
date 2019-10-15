package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

func init() {
	schemaCmd.AddCommand(pgSchemaCmd)
}

var pgSchemaCmd = &cobra.Command{
	Use:   "pg",
	Short: "Creates postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		if len(databaseURL) > 0 {
			log.Println("creating postgres schema...")
		}

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:   apiURL,
			SpaceID:  spaceID,
			CdnToken: cdnToken,
			CmaURL:   cmaURL,
			CmaToken: cmaToken,
		})

		log.Println("get space...")
		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get space done")

		log.Println("get cma types...")
		cmaTypes, err := client.ContentTypes.GetCMATypes()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("get cma done")

		schema := gontentful.NewPGSQLSchema(schemaName, space, cmaTypes.Items)
		str, err := schema.Render()
		if err != nil {
			log.Fatal(err)
		}

		// ioutil.WriteFile("/tmp/schema", []byte(str), 0644)

		if len(databaseURL) == 0 {
			fmt.Println(str)
			return
		} else {
			log.Println("postgres schema successfully created")
		}

		log.Println("executing postgres schema...")

		db, _ := sqlx.Connect("postgres", databaseURL)
		txn, err := db.Beginx()
		if err != nil {
			log.Fatal(err)
		}

		// set schema in use
		_, err = txn.Exec(fmt.Sprintf("SET search_path='%s'", schemaName))
		if err != nil {
			log.Fatal(err)
		}

		// ioutil.WriteFile("/tmp/schema", []byte(str), 0644)

		_, err = txn.Exec(str)
		if err != nil {
			log.Fatal(err)
		}

		err = txn.Commit()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("creating postgres functions...")

		funcs := gontentful.NewPGFunctions(schemaName)
		err = funcs.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("postgres schema successfully executed")
	},
}
