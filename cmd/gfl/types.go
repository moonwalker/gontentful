package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/moonwalker/gontentful"

	"github.com/spf13/cobra"
)

var typesCmd = &cobra.Command{
	Use:   "types",
	Short: "List all content types",

	Run: func(cmd *cobra.Command, args []string) {
		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL: "cdn.contentful.com",
			//SpaceID:  "dbq0oal15rwl",
			//CdnToken: "760b2a53f6b785f745e72f087b266e0c3feeb2f203e77dbcc69b8eeaa2922c14",
			SpaceID:  SpaceId,
			CdnToken: CdnToken,
		})

		data, err := client.ContentTypes.Get()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		resp := &Resp{}
		err = json.Unmarshal(data, resp)

		for _, item := range resp.Items {
			fmt.Println(item.Sys.ID)
			for _, field := range item.Fields {
				fmt.Println("- " + field.ID)
				if field.Items != nil && len(field.Items.Validations) > 0 && len(field.Items.Validations[0].LinkContentType) > 0 {
					fmt.Println("--- " + field.Items.Validations[0].LinkContentType[0])
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(typesCmd)
}
