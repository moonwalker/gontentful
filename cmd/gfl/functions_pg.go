package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	storeToFile  bool
	includeDepth int64
)

func init() {
	funcCmd.PersistentFlags().BoolVarP(&storeToFile, "file", "f", false, "store to file")
	funcCmd.PersistentFlags().Int64VarP(&includeDepth, "include", "i", 3, "include depth")
	funcCmd.AddCommand(pgFuncCmd)
}

var pgFuncCmd = &cobra.Command{
	Use:   "pg",
	Short: "Create or replace postgres functions",

	Run: func(cmd *cobra.Command, args []string) {
		dat, err := os.ReadFile("/tmp/schema")
		if err == nil {
			res := &gontentful.PGSQLSchema{}
			err = json.Unmarshal(dat, &res)
			if err == nil {
				log.Println("creating or replacing functions from cached schema...")
				funcs := gontentful.NewPGFunctions(res)
				err = funcs.Exec(databaseURL)
				if err != nil {
					log.Fatal(err)
				}
				log.Println("exec done")
				return
			}
		}

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:        apiURL,
			SpaceID:       spaceID,
			EnvironmentID: environmentID,
			CdnToken:      cdnToken,
			CmaURL:        cmaURL,
			CmaToken:      cmaToken,
		})

		log.Println("get space...")
		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("get cma types...")
		cmaTypes, err := client.ContentTypes.GetCMATypes()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("creating postgres schema...")
		schema := gontentful.NewPGSQLSchema(schemaName, space.Locales, "", cmaTypes.Items, includeDepth)

		if storeToFile {
			s, err := json.Marshal(schema)
			if err != nil {
				log.Print(err)
			} else {
				os.WriteFile("/tmp/schema", s, 0644)
			}
		}

		log.Println("creating or replacing functions...")
		funcs := gontentful.NewPGFunctions(schema)
		err = funcs.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("exec done")
	},
}
