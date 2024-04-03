package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/moonwalker/gontentful"
)

var (
	tableName, sysID string
)

func init() {
	pgDeleteCmd.PersistentFlags().StringVarP(&tableName, "table", "e", "", "table name")
	pgDeleteCmd.PersistentFlags().StringVarP(&sysID, "sys", "i", "", "sys id")
	deleteCmd.AddCommand(pgDeleteCmd)
}

var pgDeleteCmd = &cobra.Command{
	Use:   "pg",
	Short: "Delete content by content type and sys id",

	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("deleting %s from %s.%s...", sysID, schemaName, tableName)
		query := gontentful.NewPGDelete(schemaName, &gontentful.Sys{
			ID: sysID,
			ContentType: &gontentful.ContentType{
				Sys: &gontentful.Sys{
					ID: tableName,
				},
			},
		})
		txn, err := getTransaction(databaseURL, schemaName)
		if err != nil {
			log.Fatal(err)
		}
		err = query.Exec(databaseURL, txn)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("content %s deleted successfully", schemaName)
	},
}
