package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/go-github/v48/github"
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

func GetCMSEntries(contentType string, repo string, include int) (*Entries, *ContentTypes, error) {
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
				log.Fatal(fmt.Errorf("failed to format file content: %s", err.Error()))
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
			log.Fatal(fmt.Errorf("failed to fetch includes list: %s", err.Error()))
		}
		entries.Includes.Entry = includedEntries
	}

	cts := make([]*ContentType, 0)
	for _, schema := range schemas {
		if schema.ID == "_asset" {
			continue
		}
		t, err := FormatSchema(schema)
		if err != nil {
			log.Fatal(fmt.Sprintf("Failed to format schema: %s", err.Error()))
		}
		cts = append(cts, t)
	}
	contentTypes := &ContentTypes{
		Total: len(cts),
		Items: cts,
	}

	// tmp debug
	/*b, err := json.Marshal(*entries)
	if err != nil {
		log.Fatal(err.Error())
	}
	ioutil.WriteFile("/tmp/entries.json", b, 0644)*/

	return entries, contentTypes, nil
}

func getContentLocalized_old(repo string, ct string) (map[string]*content.Schema, map[string]map[string]map[string]content.ContentData) {
	ctx := context.Background()
	cfg := getConfig(ctx, accessToken, owner, repo, branch)
	localizedData := make(map[string]map[string]map[string]content.ContentData)
	schemas := make(map[string]*content.Schema)

	path := filepath.Join(cfg.WorkDir, ct)
	rcs, _, err := gh.GetContentsRecursive_old(ctx, accessToken, owner, repo, branch, path)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get json(s) from github: %s", err.Error()))
	}

	for _, rc := range rcs {
		ect := extractContentype(*rc.Path)
		if localizedData[ect] == nil {
			localizedData[ect] = make(map[string]map[string]content.ContentData)
		}
		ld := localizedData[ect]

		ghc, err := rc.GetContent()
		if err != nil {
			log.Fatal(fmt.Errorf("RepositoryContent.GetContent failed: %s", err.Error()))
		}

		if *rc.Name == content.JsonSchemaName {
			m := &content.Schema{}
			err = json.Unmarshal([]byte(ghc), m)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to unmarshal schema %s: %s", ect, err.Error()))
			}
			schemas[ect] = m
			continue
		}

		id, loc, err := parseFileName(*rc.Name)
		if err != nil {
			//fmt.Println(fmt.Sprintf("Skipping file: %s Err: %s", *rc.Path, err.Error()))
			continue
		}

		data := content.ContentData{}
		err = json.Unmarshal([]byte(ghc), &data)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to unmarshal file content(%s): %s", *rc.Path, err.Error()))
		}

		if ld[id] == nil {
			ld[id] = make(map[string]content.ContentData)
		}
		ld[id][loc] = data
	}

	return schemas, localizedData
}

func getContentLocalized(repo string, ct string) (map[string]*content.Schema, map[string]map[string]map[string]content.ContentData) {
	ctx := context.Background()
	cfg := getConfig(ctx, accessToken, owner, repo, branch)
	localizedData := make(map[string]map[string]map[string]content.ContentData)
	schemas := make(map[string]*content.Schema)

	path := filepath.Join(cfg.WorkDir, ct)
	var rcs []*github.RepositoryContent
	var err error
	if len(ct) == 0 {
		rcs, _, err = gh.GetArchivedContents(ctx, accessToken, owner, repo, branch, path)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to get json(s) from github: %s", err.Error()))
		}
	} else {
		rcs, _, err = gh.GetContentsRecursive(ctx, accessToken, owner, repo, branch, path)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to get json(s) from github: %s", err.Error()))
		}
	}

	for _, rc := range rcs {
		ect := extractContentype(*rc.Path)
		if localizedData[ect] == nil {
			localizedData[ect] = make(map[string]map[string]content.ContentData)
		}
		ld := localizedData[ect]

		if *rc.Name == content.JsonSchemaName {
			m := &content.Schema{}
			err = json.Unmarshal([]byte(*rc.Content), m)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to unmarshal schema %s: %s", ect, err.Error()))
			}
			schemas[ect] = m
			continue
		}

		id, loc, err := parseFileName(*rc.Name)
		if err != nil {
			//fmt.Println(fmt.Sprintf("Skipping file: %s Err: %s", *rc.Path, err.Error()))
			continue
		}

		data := content.ContentData{}
		err = json.Unmarshal([]byte(*rc.Content), &data)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to unmarshal file content(%s): %s", *rc.Path, err.Error()))
		}

		if ld[id] == nil {
			ld[id] = make(map[string]content.ContentData)
		}
		ld[id][loc] = data
	}

	return schemas, localizedData
}

func GetLocalContentsRecursive(path string) ([]*github.RepositoryContent, error) {
	resp := make([]*github.RepositoryContent, 0)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if f.IsDir() {
			fs, err := GetLocalContentsRecursive(fmt.Sprintf("%s/%s", path, f.Name()))
			if err != nil {
				return nil, err
			}
			resp = append(resp, fs...)
		} else {
			fName := f.Name()
			fPath := fmt.Sprintf("%s/%s", path, f.Name())
			fc, err := ioutil.ReadFile(fPath)
			if err != nil {
				return nil, err
			}
			fContent := string(fc)
			rc := &github.RepositoryContent{
				Name:    &fName,
				Path:    &fPath,
				Content: &fContent,
			}
			resp = append(resp, rc)
		}
	}

	return resp, nil
}

func formatIncludesRecursive(repo string, entryRefs map[string]string, include int, schemas map[string]*content.Schema, localizedData map[string]map[string]map[string]content.ContentData) ([]*Entry, error) {
	includes := make([]*Entry, 0)

	include--

	for id, ct := range entryRefs {
		// fetch schema and data if needed
		if schemas[ct] == nil {
			isc, ild := getContentLocalized(repo, ct)
			if isc[ct] == nil || ild[ct] == nil {
				fmt.Println(fmt.Sprintf("content %s was not found", ct))
				continue
			}

			mergeMaps(schemas, isc)
			mergeMaps(localizedData, ild)
		}

		e, er, err := FormatData(ct, id, schemas, localizedData)
		if err != nil {
			log.Fatal(fmt.Errorf("failed to format file content: %s", err.Error()))
		}

		includes = append(includes, e)
		if len(er) > 0 && include > 0 {
			en, err := formatIncludesRecursive(repo, er, include, schemas, localizedData)
			if err != nil {
				return nil, err
			}
			includes = append(includes, en...)
		}
	}

	return includes, nil
}
