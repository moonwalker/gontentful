package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/moonwalker/gontentful"
	"github.com/spf13/cobra"
)

var (
	contentTypeMigrateCmd = &cobra.Command{
		Use:   "migrateContentType",
		Short: "migrate content type",
		PreRun: func(cmd *cobra.Command, args []string) {
			if direction == "toContentful" {
				cmd.MarkPersistentFlagRequired("repo")
			} else {
				rootCmd.MarkPersistentFlagRequired("space")
				rootCmd.MarkPersistentFlagRequired("token")
				rootCmd.MarkPersistentFlagRequired("cma")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			switch direction {
			case "toContentful":
				formatContentType()
			case "fromContentful":
				transformContentType()
			}
		},
	}
)

func init() {
	// contentType to migrate
	contentTypeMigrateCmd.Flags().StringVarP(&contentType, "contentModel", "m", "", "type of the content to migrate")
	contentTypeMigrateCmd.Flags().StringVarP(&repo, "repo", "r", "", "repo of the content to migrate")
	contentTypeMigrateCmd.PersistentFlags().StringVarP(&direction, "direction", "d", "", "directions: <fromContentful|toContentful>")
	contentTypeMigrateCmd.MarkPersistentFlagRequired("direction")
	rootCmd.AddCommand(contentTypeMigrateCmd)
}

func transformContentType() {
	fmt.Println("ContentType migration started")

	opts := &gontentful.ClientOptions{
		SpaceID:       spaceID,
		EnvironmentID: "master",
		CdnToken:      cdnToken,
		CdnURL:        "cdn.contentful.com",
		CmaToken:      cmaToken,
		CmaURL:        "api.contentful.com",
	}
	cli := gontentful.NewClient(opts)

	var err error
	var types *gontentful.ContentTypes
	var data []byte
	if len(contentType) > 0 {
		data, err = cli.ContentTypes.GetSingleCMA(contentType)
		if err == nil {
			ct := &gontentful.ContentType{}
			err = json.Unmarshal(data, ct)
			if err == nil {
				types = &gontentful.ContentTypes{
					Total: 1,
					Items: []*gontentful.ContentType{ct},
				}
			}
		}
	} else {
		types, err = cli.ContentTypes.GetCMATypes()
	}
	if err != nil {
		log.Fatal(errors.New(fmt.Sprintf("failed to fetch content type(s): %s", err.Error())))
	}

	if types.Total > 0 {
		for _, item := range types.Items {
			schema, err := gontentful.TransformModel(item)
			if err != nil {
				log.Fatal(errors.New(fmt.Sprintf("failed to transform model: %s", err.Error())))
			}
			fmt.Println(fmt.Sprintf("SchemaName: %+v", *schema))
		}
	}

	fmt.Println("CotentType successfully migrated")
}

func formatContentType() {
	fmt.Println("ContentType formatting started")

	fmt.Println("ContentType successfully formatted")
}
