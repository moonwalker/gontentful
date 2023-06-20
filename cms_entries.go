package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
)

func GetCMSEntries(contentType string, repo string, include int) (*Entries, *ContentTypes, error) {
	schemas, localizedData, err := getContentLocalized(repo, contentType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get content localized: %s", err.Error())
	}

	entries, err := createEntriesFromLocalizedData(repo, schemas, localizedData, include)

	cts := make([]*ContentType, 0)
	for _, schema := range schemas {
		if schema.ID == ASSET_TABLE_NAME {
			continue
		}
		cts = append(cts, formatSchema(schema))
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

func createEntriesFromLocalizedData(repo string, schemas map[string]*content.Schema, localizedData map[string]map[string]map[string]content.ContentData, include int) (*Entries, error) {
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
				return nil, fmt.Errorf("failed to format file content: %s", err.Error())
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

		includedEntries, includedAssets, err := formatIncludesRecursive(repo, includes, include, schemas, localizedData)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch includes list: %s", err.Error())
		}
		entries.Includes.Entry = includedEntries
		entries.Includes.Asset = includedAssets
	}

	return entries, nil
}

func GetCMSEntry(contentType string, repo string, prefix string, include int) (*Entries, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, contentType)

	rcs, _, err := gh.GetAllLocaleContentsWithTree(ctx, cfg.Token, owner, repo, branch, path, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get all localized contents: %s", err.Error())
	}

	schemas, localizedData, err := formatRepositoryContent(rcs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to format repository content: %s", err.Error())
	}

	entries, err := createEntriesFromLocalizedData(repo, schemas, localizedData, include)
	if err != nil {
		return nil, fmt.Errorf("failed to format repository content: %s", err.Error())
	}

	return entries, nil
}

func GetPublishedEntry(repo string, contentType string, prefix string) (*PublishedEntry, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, contentType)

	rcs, _, err := gh.GetAllLocaleContentsWithTree(ctx, cfg.Token, owner, repo, branch, path, prefix)
	//rcs, _, err := gh.GetAllLocaleContents(ctx, cfg.Token, owner, repo, branch, path, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get all localized contents: %s", err.Error())
	}

	// get model schema
	refFields := make(map[string]*content.Field, 0)
	for _, rc := range rcs {
		if *rc.Name == content.JsonSchemaName {
			schema := &content.Schema{}
			err = json.Unmarshal([]byte(*rc.Content), schema)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal schema %s: %s", contentType, err.Error())
			}
			for _, sf := range schema.Fields {
				if sf.Reference {
					refFields[sf.ID] = sf
				}
			}
		}
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
			return nil, fmt.Errorf("failed to unmarshal content data: %s", err.Error())
		}
		for k, v := range data.Fields {
			if fields[k] == nil {
				fields[k] = make(map[string]interface{})
			}
			if rf := refFields[k]; rf != nil {
				if fields[k] == nil {
					fields[k] = make(map[string]interface{})
				}
				if rf.List {
					if rl, ok := v.([]interface{}); ok {
						refList := make([]interface{}, 0)
						for _, r := range rl {
							if rid, ok := r.(string); ok {
								esys := make(map[string]interface{})
								esys["type"] = LINK
								esys["linkType"] = ENTRY
								esys["id"] = rid
								es := make(map[string]interface{})
								es["sys"] = esys
								refList = append(refList, es)
							}
						}
						fields[k][loc] = refList
					}
				} else {
					esys := make(map[string]interface{})
					esys["type"] = LINK
					esys["linkType"] = ENTRY
					esys["id"] = v
					es := make(map[string]interface{})
					es["sys"] = esys
					fields[k][loc] = es
				}
			} else {
				fields[k][loc] = v
			}
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

	path := filepath.Join(cfg.WorkDir, ct)
	var rcs []*github.RepositoryContent
	var err error
	/*if len(ct) == 0 || ct != ASSET_TABLE_NAME {
		//localPath := fmt.Sprintf("/Users/rolandhelli/Development/src/github.com/moonwalker/%s", repo)
		//rcs, err = GetLocalContentsRecursive(localPath)
		rcs, _, err = gh.GetArchivedContents(ctx, cfg.Token, owner, repo, branch, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get json(s) from github: %s", err.Error())
		}
	} else {
		rcs, _, err = gh.GetContentsRecursive(ctx, cfg.Token, owner, repo, branch, path)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get json(s) from github: %s", err.Error())
		}
	}*/

	// test, use GetArchivedContents in all use cases
	rcs, _, err = gh.GetArchivedContents(ctx, cfg.Token, owner, repo, branch, path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get json(s) from github: %s", err.Error())
	}

	schemas, localizedData, err := formatRepositoryContent(rcs, path)

	return schemas, localizedData, err
}

func formatRepositoryContent(rcs []*github.RepositoryContent, path string) (map[string]*content.Schema, map[string]map[string]map[string]content.ContentData, error) {
	localizedData := make(map[string]map[string]map[string]content.ContentData)
	schemas := make(map[string]*content.Schema)
	var err error

	for _, rc := range rcs {
		ect := extractContentype(filepath.Join(path, *rc.Path))
		if ect == IMAGE_FOLDER_NAME {
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
			fmt.Println("unmarshal err", *rc.Content)
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
		return nil, fmt.Errorf("failed to read dir at %s: %s", path, err.Error())
	}
	for _, f := range files {
		if f.IsDir() {
			fs, err := GetLocalContentsRecursive(fmt.Sprintf("%s/%s", path, f.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to get local contents recursive: %s", err.Error())
			}
			resp = append(resp, fs...)
		} else {
			fName := f.Name()
			fPath := fmt.Sprintf("%s/%s", path, f.Name())
			fc, err := ioutil.ReadFile(fPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read file at %s: %s", path, err.Error())
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

func formatIncludesRecursive(repo string, entryRefs map[string]string, include int, schemas map[string]*content.Schema, localizedData map[string]map[string]map[string]content.ContentData) ([]*Entry, []*Entry, error) {
	includes := make([]*Entry, 0)
	assets := make([]*Entry, 0)

	include--

	for id, ct := range entryRefs {
		// fetch schema and data if needed
		if schemas[ct] == nil {
			isc, ild, err := getContentLocalized(repo, ct)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get format localized: %s", err.Error())
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
			return nil, nil, fmt.Errorf("failed to format file content: %s", err.Error())
		}

		if ct == ASSET_TABLE_NAME {
			assets = append(assets, e)
		} else {
			includes = append(includes, e)
		}
		if len(er) > 0 && include > 0 {
			en, as, err := formatIncludesRecursive(repo, er, include, schemas, localizedData)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get format includes recursive: %s", err.Error())
			}
			includes = append(includes, en...)
			assets = append(includes, as...)
		}
	}

	return includes, assets, nil
}
