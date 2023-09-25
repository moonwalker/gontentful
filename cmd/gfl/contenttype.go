package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/moonwalker/gontentful"
	"github.com/moonwalker/moonbase/pkg/content"
)

func transformContentType() {
	start := time.Now()
	fmt.Println("ContentType transforming started")

	opts := &gontentful.ClientOptions{
		SpaceID:       spaceID,
		EnvironmentID: "master",
		CdnToken:      cdnToken,
		CdnURL:        apiURL,
		CmaToken:      cmaToken,
		CmaURL:        cmaURL,
	}
	cli := gontentful.NewClient(opts)

	var err error
	var types *gontentful.ContentTypes
	if len(contentType) > 0 {
		var ctype *gontentful.ContentType
		ctype, err = cli.ContentTypes.GetSingleCMA(contentType)
		if err == nil {
			types = &gontentful.ContentTypes{
				Total: 1,
				Items: []*gontentful.ContentType{ctype},
			}
		}
	} else {
		types, err = cli.ContentTypes.GetCMATypes()
	}
	if err != nil {
		log.Fatalf("failed to fetch content type(s): %s", err.Error())
	}
	if types.Total == 0 {
		log.Fatalf("contenttype(s) not found: %s", contentType)
	}

	os.RemoveAll(fmt.Sprintf(outputFormat, ""))

	for _, item := range types.Items {
		schema, err := gontentful.TransformModel(item)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to transform model: %s", err.Error()))
		}
		path := fmt.Sprintf(outputFormat, item.Sys.ID)
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to create output folder %s: %s", path, err.Error()))
		}
		b, _ := json.Marshal(schema)
		f := fmt.Sprintf("%s/%s", path, content.JsonSchemaName)
		fmt.Printf("Writing file: %s", f)
		os.WriteFile(f, b, 0644)
		fmt.Printf("\033[2K")
		fmt.Println()
		fmt.Printf("\033[1A")
	}
	// _assets schema
	aid := gontentful.ASSET_TABLE_NAME
	schema := &content.Schema{
		ID:   aid,
		Name: "Asset",
		Fields: []*content.Field{
			{
				ID:    "file",
				Label: "File",
				Type:  "json",
			},
			{
				ID:    "title",
				Label: "Title",
				Type:  "text",
			},
		},
	}
	path := fmt.Sprintf(outputFormat, aid)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create output folder %s: %s", path, err.Error())
	}
	b, _ := json.Marshal(schema)
	f := fmt.Sprintf("%s/%s", path, content.JsonSchemaName)
	fmt.Printf("Writing file: %s", f)
	os.WriteFile(f, b, 0644)
	fmt.Printf("\033[2K")
	fmt.Println()
	fmt.Printf("\033[1A")
	fmt.Printf("ContentType successfully transformed in %.1fss\n", time.Since(start).Seconds())
}

func formatContentType() {
	fmt.Println(fmt.Sprintf("ContentType formatting started on:%s|%s", repo, contentType))

	cts, err := gontentful.GetCMSSchemas(repo, contentType)
	if err != nil {
		log.Fatalf("Error in GetCMSSchemas: %s", err.Error())
	}

	fmt.Println(fmt.Sprintf("%v schemas successfully formatted for content sync.", len(cts.Items)))
}

func readDir(path string) []fs.DirEntry {
	dirEntry, err := os.ReadDir(path)
	if err != nil {
		log.Fatalf("Failed to read input directory: %s", err.Error())
	}

	return dirEntry
}
