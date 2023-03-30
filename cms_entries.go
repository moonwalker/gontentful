package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/google/go-github/v48/github"
	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

type Config struct {
	Token   string
	WorkDir string `json:"workdir" yaml:"workdir"`
}

const (
	queryLimit = 1000
	owner      = "moonwalker"
	branch     = "main"
	configPath = "moonbase.yaml"
	include    = 0
)

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
				log.Fatalf("failed to format file content: %s", err.Error())
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
			log.Fatalf("failed to fetch includes list: %s", err.Error())
		}
		entries.Includes.Entry = includedEntries
	}

	cts := make([]*ContentType, 0)
	for _, schema := range schemas {
		if schema.ID == ASSET_TABLE_NAME {
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

func GetPublishedEntry(repo string, contentType string, prefix string) (*PublishedEntry, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, contentType)

	rcs, _, err := gh.GetAllLocaleContents(ctx, cfg.Token, owner, repo, branch, path, prefix)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]map[string]interface{})
	var sys *Sys
	for _, rc := range rcs {
		_, loc, err := parseFileName(*rc.Name)
		if err != nil {
			//fmt.Println(fmt.Sprintf("Skipping file: %s Err: %s", *rc.Path, err.Error()))
			continue
		}
		data := content.ContentData{}
		err = json.Unmarshal([]byte(*rc.Content), &data)
		if err != nil {
			return nil, err
		}
		for k, v := range data.Fields {
			if fields[k] == nil {
				fields[k] = make(map[string]interface{})
			}
			fields[k][loc] = v
		}
		if sys == nil {
			sys = &Sys{
				ID:        data.ID,
				CreatedAt: data.CreatedAt.Format(time.RFC3339Nano),
				UpdatedAt: data.UpdatedAt.Format(time.RFC3339Nano),
				Version:   data.Version,
			}
		}
	}
	return &PublishedEntry{
		Sys:    sys,
		Fields: PublishFields(fields),
	}, nil
}

func getContentLocalized(repo string, ct string) (map[string]*content.Schema, map[string]map[string]map[string]content.ContentData, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	localizedData := make(map[string]map[string]map[string]content.ContentData)
	schemas := make(map[string]*content.Schema)

	path := filepath.Join(cfg.WorkDir, ct)
	var rcs []*github.RepositoryContent
	var err error
	if len(ct) == 0 {
		rcs, _, err = gh.GetArchivedContents(ctx, cfg.Token, owner, repo, branch, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get json(s) from github: %s", err.Error())
		}
	} else {
		rcs, _, err = gh.GetContentsRecursive(ctx, cfg.Token, owner, repo, branch, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get json(s) from github: %s", err.Error())
		}
	}

	for _, rc := range rcs {
		ect := extractContentype(*rc.Path)
		if ect == content.ImageFolder {
			continue
		}
		if localizedData[ect] == nil {
			localizedData[ect] = make(map[string]map[string]content.ContentData)
		}
		ld := localizedData[ect]

		if *rc.Name == content.JsonSchemaName {
			m := &content.Schema{}
			err = json.Unmarshal([]byte(*rc.Content), m)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal schema %s: %s", ect, err.Error())
			}
			schemas[ect] = m
			continue
		}

		_, loc, err := parseFileName(*rc.Name)
		if err != nil {
			//fmt.Println(fmt.Sprintf("Skipping file: %s Err: %s", *rc.Path, err.Error()))
			continue
		}

		data := content.ContentData{}
		err = json.Unmarshal([]byte(*rc.Content), &data)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal file content(%s): %s", *rc.Path, err.Error())
		}

		if ld[data.ID] == nil {
			ld[data.ID] = make(map[string]content.ContentData)
		}
		ld[data.ID][loc] = data
	}

	return schemas, localizedData, nil
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
			isc, ild, err := getContentLocalized(repo, ct)
			if err != nil {
				return nil, fmt.Errorf("failed to get format localized: %s", err.Error())
			}
			if isc[ct] == nil || ild[ct] == nil {
				fmt.Println(fmt.Sprintf("content %s was not found", ct))
				continue
			}

			mergeMaps(schemas, isc)
			mergeMaps(localizedData, ild)
		}

		e, er, err := FormatData(ct, id, schemas, localizedData)
		if err != nil {
			return nil, fmt.Errorf("failed to format file content: %s", err.Error())
		}

		includes = append(includes, e)
		if len(er) > 0 && include > 0 {
			en, err := formatIncludesRecursive(repo, er, include, schemas, localizedData)
			if err != nil {
				return nil, fmt.Errorf("failed to get format includes recursive: %s", err.Error())
			}
			includes = append(includes, en...)
		}
	}

	return includes, nil
}
