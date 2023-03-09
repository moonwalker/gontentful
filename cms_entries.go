package gontentful

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

type Config struct {
	WorkDir string `json:"workdir" yaml:"workdir"`
}

const (
	queryLimit = 1000
	owner      = "moonwalker"
	branch     = "main"
	configPath = "moonbase.yaml"
	include    = 0
)

var accessToken = os.Getenv("GITHUB_TOKEN")

func GetCMSEntries(contentType string, repo string, include int) (*Entries, error) {
	schemas, localizedData := getContentLocalized(repo, contentType)
	entries := &Entries{
		Sys: &Sys{
			Type: "Array",
		},
	}

	includes := make(map[string]string)
	for ct, locData := range localizedData {
		for id, _ := range locData {
			entry, entryRefs, err := FormatData(ct, id, schemas, localizedData)
			if err != nil {
				log.Fatal(errors.New(fmt.Sprintf("failed to format file content: %s", err.Error())))
			}
			mergeMaps(includes, entryRefs)
			entries.Items = append(entries.Items, entry)
		}
	}
	entries.Total = len(entries.Items)

	if include > 0 && len(includes) > 0 {
		if entries.Includes == nil {
			entries.Includes = &Include{}
		}

		includedEntries, err := formatIncludesRecursive(repo, includes, include, schemas, localizedData)
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("failed to fetch includes list: %s", err.Error())))
		}
		entries.Includes.Entry = includedEntries
	}

	// tmp debug
	/*b, err := json.Marshal(*entries)
	if err != nil {
		log.Fatal(err.Error())
	}
	ioutil.WriteFile("/tmp/entries.json", b, 0644)*/

	return entries, nil
}

func getContentLocalized(repo string, ct string) (map[string]*content.Schema, map[string]map[string]map[string]content.ContentData) {
	ctx := context.Background()
	cfg := getConfig(ctx, accessToken, owner, repo, branch)
	localizedData := make(map[string]map[string]map[string]content.ContentData)
	schemas := make(map[string]*content.Schema)

	path := filepath.Join(cfg.WorkDir, ct)
	rcs, _, err := gh.GetContentsRecursive(ctx, accessToken, owner, repo, branch, path)
	if err != nil {
		log.Fatal(errors.New(fmt.Sprintf("failed to get json(s) from github: %s", err.Error())))
	}

	for _, rc := range rcs {
		ect := extractContentype(*rc.Path)
		if localizedData[ect] == nil {
			localizedData[ect] = make(map[string]map[string]content.ContentData)
		}
		ld := localizedData[ect]

		ghc, err := rc.GetContent()
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("RepositoryContent.GetContent failed: %s", err.Error())))
		}

		if *rc.Name == content.JsonSchemaName {
			m := &content.Schema{}
			err = json.Unmarshal([]byte(ghc), m)
			if err != nil {
				log.Fatal((errors.New((fmt.Sprintf("failed to unmarshal schema %s: %s", ect, err.Error())))))
			}
			schemas[ect] = m
			continue
		}

		id, loc, err := parseFileName(*rc.Name)
		if err != nil {
			fmt.Println(fmt.Sprintf("Skipping file: %s Err: %s", *rc.Path, err.Error()))
			continue
		}

		data := content.ContentData{}
		err = json.Unmarshal([]byte(ghc), &data)
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("failed to unmarshal file content(%s): %s", *rc.Path, err.Error())))
		}

		if ld[id] == nil {
			ld[id] = make(map[string]content.ContentData)
		}
		ld[id][loc] = data
	}

	return schemas, localizedData
}

func formatIncludesRecursive(repo string, entryRefs map[string]string, include int, schemas map[string]*content.Schema, localizedData map[string]map[string]map[string]content.ContentData) ([]*Entry, error) {
	includes := make([]*Entry, 0)

	include--

	for id, ct := range entryRefs {
		if ct == "_asset" {
			continue
		}
		e, er, err := FormatData(ct, id, schemas, localizedData)
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("failed to format file content: %s", err.Error())))
		}
		includes = append(includes, e)
		if len(er) > 0 && include > 0 {
			// fetch schema and data if needed
			for _, ct := range er {
				if schemas[ct] == nil {
					isc, ild := getContentLocalized(repo, ct)
					mergeMaps(schemas, isc)
					mergeMaps(localizedData, ild)
				}
			}

			en, err := formatIncludesRecursive(repo, er, include, schemas, localizedData)
			if err != nil {
				return nil, err
			}
			includes = append(includes, en...)
		}
	}

	return includes, nil
}
