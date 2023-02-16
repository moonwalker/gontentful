package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"

	"github.com/moonwalker/gontentful"
	"github.com/moonwalker/moonbase/pkg/content"
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
			path := fmt.Sprintf("./output/%s", item.Sys.ID)
			err = os.MkdirAll(path, os.ModePerm)
			if err != nil {
				log.Fatal(errors.New(fmt.Sprintf("failed to create output folder %s: %s", path, err.Error())))
			}
			b, err := json.Marshal(schema)
			fmt.Println(fmt.Sprintf("Writing file: %s/_schema.json", path))
			ioutil.WriteFile(fmt.Sprintf("%s/_schema.json", path), b, 0644)
		}
	}
	fmt.Println("ContentType successfully migrated")
}

func formatContentType() {
	fmt.Println("ContentType formatting started")
	sPath := "./input"

	paths := make([]string, 0)
	if len(contentType) > 0 {
		paths = append(paths, fmt.Sprintf("%s/%s/_schema.json", sPath, contentType))
	} else {
		p := readDir(fmt.Sprintf("%s", sPath))
		for _, d := range p {
			if d.IsDir() {
				paths = append(paths, fmt.Sprintf("%s/%s/_schema.json", sPath, d.Name()))
			}
		}
	}

	for _, s := range paths {
		f, err := ioutil.ReadFile(s)
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("Failed to read file %s: %s", s, err.Error())))
		}
		m := content.Schema{}
		_ = json.Unmarshal([]byte(f), &m)
		t, err := gontentful.FormatSchema(&m)
		if err != nil {
			log.Fatal((errors.New((fmt.Sprintf("Error formatting gontenful schema %s: %s", s, err.Error())))))
		}
		// TODO: Do something with the formatted schema - push to contentful??
		// For testing:
		b, err := json.Marshal(t)
		if err != nil {
			fmt.Println("Error marshalling schema for testing.", err.Error())
		}
		ioutil.WriteFile(fmt.Sprintf("%s/%s/test.json", sPath, t.Name), b, 0644)
		fmt.Println("Schema to push to contentful: ", *t)
	}

	fmt.Println("ContentType successfully formatted")
}

func readDir(path string) []fs.DirEntry {
	dirEntry, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(errors.New(fmt.Sprintf("Failed to read input directory: %s", err.Error())))
	}

	return dirEntry
}
