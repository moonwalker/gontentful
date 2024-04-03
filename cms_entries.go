package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

var localeVariants = map[string]string{
	"nb": "no",
	"no": "nb",
}

func GetCMSEntries(contentType string, repo string, include int) (*Entries, *ContentTypes, error) {
	schemas, localizedData, err := getContentLocalized(repo, contentType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get content localized: %s", err.Error())
	}
	entries, err := createEntriesFromLocalizedData(repo, schemas, localizedData, include)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create loclalized entires: %s", err.Error())
	}

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
	os.WriteFile("/tmp/entries.json", b, 0644)*/

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
		for id := range locData {
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

func GetCMSEntry(contentType string, repo string, name string, locales []*Locale, include int) (*Entries, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, contentType)

	/*files := make([]string, 0)
	for _, l := range locales {
		files = append(files, fmt.Sprintf("%s/%s.json", name, l.Code))
	}
	files = append(files, content.JsonSchemaName)
	rcs, _, err := gh.GetFilesContent(ctx, cfg.Token, owner, repo, branch, path, files)
	if err != nil {
		return nil, fmt.Errorf("failed to get all localized contents: %s", err.Error())
	}*/

	// get contentType's schema
	rcs := make([]*github.RepositoryContent, 0)
	rcSchema, _, err := gh.GetSchema(ctx, cfg.Token, owner, repo, branch, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema(%s): %s", contentType, err.Error())
	}
	sc, err := rcSchema.GetContent()
	if err != nil {
		return nil, fmt.Errorf("RepositoryContent.GetContent failed: %s", err.Error())
	}
	rcSchema.Content = &sc
	rcs = append(rcs, rcSchema)

	// get all localized jsons
	itemPath := fmt.Sprintf("%s/%s", path, name)
	rcsAll, _, err := gh.GetContentsRecursive(ctx, cfg.Token, owner, repo, branch, itemPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get all localized contents: %s", err.Error())
	}
	// helper map
	contentLocMap := make(map[string]*github.RepositoryContent)
	for _, rc := range rcsAll {
		c, err := rc.GetContent()
		if err != nil {
			return nil, fmt.Errorf("RepositoryContent.GetContent failed: %s", err.Error())
		}
		rc.Content = &c
		loc, _ := extractFileInfo(*rc.Name)
		contentLocMap[loc] = rc
	}

	// create response array
	for _, l := range locales {
		if contentLocMap[l.Code] != nil {
			rcs = append(rcs, contentLocMap[l.Code])
		} else {
			if len(localeVariants[l.Code]) > 0 {
				lv := localeVariants[l.Code]
				if contentLocMap[lv] != nil {
					newPath := fmt.Sprintf("%s.json", l.Code)
					newName := newPath
					rcs = append(rcs, &github.RepositoryContent{
						Name:    &newName,
						Path:    &newPath,
						Content: contentLocMap[lv].Content,
					})
				}
			} else {
				fmt.Printf("failed to get '%s' localized content for %s\n", l.Code, itemPath)
			}
		}
	}

	schemas, localizedData, err := formatRepositoryContent(rcs, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to format repository content: %s", err.Error())
	}

	entries, err := createEntriesFromLocalizedData(repo, schemas, localizedData, include)
	if err != nil {
		return nil, fmt.Errorf("failed to create loclalized entires: %s", err.Error())
	}

	return entries, nil
}

func GetBlob(repo string, path string, file string) (*string, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	rcs, _, err := gh.GetFilesContent(ctx, cfg.Token, owner, repo, branch, path, []string{file})
	if err != nil {
		return nil, err
	}

	return rcs[0].Content, nil
}

func GetBlobURL(repo string, path string, file string) (string, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	rcs, _, err := gh.GetFilesContent(ctx, cfg.Token, owner, repo, branch, path, []string{file})
	if err != nil {
		return "", err
	}

	return *rcs[0].DownloadURL, nil
}

func GetPublishedEntry(repo string, contentType string, files []string) (*PublishedEntry, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, contentType)

	files = append(files, content.JsonSchemaName)
	rcs, _, err := gh.GetFilesContent(ctx, cfg.Token, owner, repo, branch, path, files)
	if err != nil {
		return nil, fmt.Errorf("failed to get all localized contents: %s", err.Error())
	}

	// get model schema
	refFields := make(map[string]*content.Field, 0)
	localizedFields := make(map[string]bool, 0)
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
				if sf.Localized {
					localizedFields[sf.ID] = true
				}
			}
		}
	}

	fields := make(map[string]map[string]interface{})
	var sys *Sys
	for _, rc := range rcs {
		if *rc.Name != content.JsonSchemaName {
			ext := filepath.Ext(*rc.Name)
			if ext != ".json" {
				continue
			}
			loc := extractLocale(*rc.Path, ext)
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
					ID:          data.ID,
					CreatedAt:   data.CreatedAt,
					UpdatedAt:   data.UpdatedAt,
					PublishedAt: data.PublishedAt,
					Version:     data.Version,
				}
			}
		}
	}

	// clear fallback values
	clearPublishedEntryFallbackValues(localizedFields, fields)

	return &PublishedEntry{
		Sys:    sys,
		Fields: PublishFields(fields),
	}, nil
}

func GetPublishedEntryFromCMSPost(repo, owner, ref, collection string, items map[string]*content.ContentData, localesArray []string) (*PublishedEntry, []*gh.BlobEntry, error) {
	return nil, nil, nil
}

func clearPublishedEntryFallbackValues(localizedFields map[string]bool, fields map[string]map[string]interface{}) {
	for fn, fvs := range fields {
		if !localizedFields[fn] {
			continue
		}
		for loc := range fvs {
			if loc != DefaultLocale && reflect.DeepEqual(fields[fn][loc], fields[fn][DefaultLocale]) {
				fields[fn][loc] = nil
			}
		}
	}
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
	if ct != "" {
		rcs, _, err = gh.GetContentsRecursive(ctx, cfg.Token, owner, repo, branch, path)
	} else {
		// test, use GetArchivedContents in all use cases
		rcs, _, err = gh.GetArchivedContents(ctx, cfg.Token, owner, repo, branch, path)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get json(s) from github: %s", err.Error())
	}

	schemas, localizedData, err := formatRepositoryContent(rcs, ct)

	return schemas, localizedData, err
}

func formatRepositoryContent(rcs []*github.RepositoryContent, contentType string) (map[string]*content.Schema, map[string]map[string]map[string]content.ContentData, error) {
	localizedData := make(map[string]map[string]map[string]content.ContentData)
	schemas := make(map[string]*content.Schema)
	var err error

	for _, rc := range rcs {
		if *rc.Name == content.JsonSchemaName {
			ect := extractContenttype(contentType, *rc.Path, 1)
			m := &content.Schema{}
			err = json.Unmarshal([]byte(*rc.Content), m)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal schema %s: %s", ect, err.Error())
			}
			schemas[ect] = m
			continue
		}

		ect := extractContenttype(contentType, *rc.Path, 2)
		if ect == IMAGE_FOLDER_NAME {
			continue
		}

		loc, ext := extractFileInfo(*rc.Name)
		if ext != ".json" {
			continue
		}

		if localizedData[ect] == nil {
			localizedData[ect] = make(map[string]map[string]content.ContentData)
		}

		data := content.ContentData{}
		err = json.Unmarshal([]byte(*rc.Content), &data)
		if err != nil {
			fmt.Println("unmarshal err", *rc.Content)
			return nil, nil, fmt.Errorf("failed to unmarshal file content(%s): %s", *rc.Path, err.Error())
		}

		if localizedData[ect][data.ID] == nil {
			localizedData[ect][data.ID] = make(map[string]content.ContentData)
		}
		if localizedData[ect][data.ID][loc].ID == data.ID {
			eRow := localizedData[ect][data.ID][loc]
			oUpdatedAt, err := time.Parse(time.RFC3339, eRow.UpdatedAt)
			swapRows := false
			if err != nil {
				swapRows = true
			} else {
				nUpdatedAt, err := time.Parse(time.RFC3339, data.UpdatedAt)
				if err == nil {
					if nUpdatedAt.After(oUpdatedAt) {
						swapRows = true
					}
				}
			}
			if swapRows {
				fmt.Println(fmt.Sprintf("duplicated SysID %s with locale %s in contentType %s swapping rows", eRow.ID, loc, ect))
				localizedData[ect][data.ID][loc] = data
			}
		} else {
			localizedData[ect][data.ID][loc] = data
		}
	}

	clearLocalizedDataFallbackValues(schemas, localizedData)

	return schemas, localizedData, nil
}

func clearLocalizedDataFallbackValues(schemas map[string]*content.Schema, localizedData map[string]map[string]map[string]content.ContentData) {
	localizedFields := make(map[string]map[string]bool)

	for ct, s := range schemas {
		localizedFields[ct] = make(map[string]bool)
		for _, sf := range s.Fields {
			if sf.Localized {
				localizedFields[ct][sf.ID] = true
			}
		}
	}

	for ct, ld := range localizedData {
		for _, d := range ld {
			clearItemFallbackValues(localizedFields[ct], d)
		}
	}
}

func clearItemFallbackValues(localizedFields map[string]bool, locData map[string]content.ContentData) {
	for loc, data := range locData {
		if loc == DefaultLocale {
			continue
		}
		for fn, _ := range data.Fields {
			if localizedFields == nil || !localizedFields[fn] {
				continue
			}
			if reflect.DeepEqual(locData[loc].Fields[fn], locData[DefaultLocale].Fields[fn]) {
				locData[loc].Fields[fn] = nil
			}
		}
	}
}

func GetLocalContentsRecursive(path string) ([]*github.RepositoryContent, error) {
	resp := make([]*github.RepositoryContent, 0)
	files, err := os.ReadDir(path)
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
			fc, err := os.ReadFile(fPath)
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
				fmt.Printf("content %s %s was not found\n", ct, id)
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
