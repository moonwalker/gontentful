package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/moonwalker/gontentful"
	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
	"gopkg.in/yaml.v3"
)

const (
	queryLimit = 1000
	owner      = "moonwalker"
	branch     = "main"
	configPath = "moonbase.yaml"
	include    = 0
)

var accessToken = os.Getenv("GITHUB_TOKEN")

type Config struct {
	WorkDir string `json:"workdir" yaml:"workdir"`
}

func transformContent() {
	fmt.Println("Content transforming started")

	opts := &gontentful.ClientOptions{
		SpaceID:       spaceID,
		EnvironmentID: "master",
		CdnToken:      cdnToken,
		CdnURL:        "cdn.contentful.com",
		CmaToken:      cmaToken,
		CmaURL:        "api.contentful.com",
	}
	cli := gontentful.NewClient(opts)

	var res *gontentful.Entries
	var err error
	if len(contentType) > 0 {
		res, err = GetContentTypeEntries(cli, contentType)
	} else {
		res, err = GetAllEntries(cli)
	}
	if err != nil {
		log.Fatal(errors.New(fmt.Sprintf("failed to fetch entries: %s", err.Error())))
	}

	locales, err := cli.Locales.GetLocales()
	if err != nil {
		log.Fatal(errors.New(fmt.Sprintf("failed to fetch locales: %s", err.Error())))
	}

	if res.Total > 0 {
		for _, item := range res.Items {
			if item.Sys.Type != "Entry" {
				continue
			}
			entries, err := gontentful.TransformEntry(locales, item)
			if err != nil {
				log.Fatal(errors.New(fmt.Sprintf("failed to transform entry: %s", err.Error())))
			}

			for l, e := range entries {
				b, err := json.Marshal(e)
				if err != nil {
					log.Fatal(errors.New(fmt.Sprintf("failed to marshal entry: %s", err.Error())))
				}
				ct := contentType
				if len(ct) == 0 {
					ct = toCamelCase(item.Sys.ContentType.Sys.ID)
				}

				path := fmt.Sprintf("./output/%s", ct)
				err = os.MkdirAll(path, os.ModePerm)
				if err != nil {
					log.Fatal(errors.New(fmt.Sprintf("failed to create output folder %s: %s", path, err.Error())))
				}

				fn := fmt.Sprintf("%s_%s", item.Sys.ID, l)
				fmt.Println(fmt.Sprintf("Writing file: %s/%s.json", path, fn))
				ioutil.WriteFile(fmt.Sprintf("%s/%s.json", path, fn), b, 0644)
			}
		}
	}

	fmt.Println("Content successfully transformed")
}

func formatContent() {
	fmt.Println("Content formatting started")

	schemas, localizedData := getContentLocalized(contentType)
	entries := &gontentful.Entries{
		Sys: &gontentful.Sys{
			Type: "Array",
		},
	}

	includes := make(map[string]string)
	for ct, locData := range localizedData {
		for id, _ := range locData {
			entry, entryRefs, err := gontentful.FormatData(ct, id, schemas, localizedData)
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
			entries.Includes = &gontentful.Include{}
		}

		includedEntries, err := formatIncludesRecursive(includes, include, schemas, localizedData)
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

	fmt.Println("Content successfully formatted")
}

func getContentLocalized(ct string) (map[string]*content.Schema, map[string]map[string]map[string]content.ContentData) {
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

func formatIncludesRecursive(entryRefs map[string]string, include int, schemas map[string]*content.Schema, localizedData map[string]map[string]map[string]content.ContentData) ([]*gontentful.Entry, error) {
	includes := make([]*gontentful.Entry, 0)

	include--

	for id, ct := range entryRefs {
		if ct == "_asset" {
			continue
		}
		e, er, err := gontentful.FormatData(ct, id, schemas, localizedData)
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("failed to format file content: %s", err.Error())))
		}
		includes = append(includes, e)
		if len(er) > 0 && include > 0 {
			// fetch schema and data if needed
			for _, ct := range er {
				if schemas[ct] == nil {
					isc, ild := getContentLocalized(ct)
					mergeMaps(schemas, isc)
					mergeMaps(localizedData, ild)
				}
			}

			en, err := formatIncludesRecursive(er, include, schemas, localizedData)
			if err != nil {
				return nil, err
			}
			includes = append(includes, en...)
		}
	}

	return includes, nil
}

func mergeMaps[M ~map[K]V, K comparable, V any](dst M, src M) {
	for k, v := range src {
		dst[k] = v
	}
}

func extractContentype(path string) string {
	items := strings.Split(path, "/")
	if len(items) > 1 {
		return items[len(items)-2]
	}
	return ""
}

func parseFileName(fn string) (string, string, error) {
	ext := filepath.Ext(fn)
	if ext != ".json" {
		return "", "", errors.New(fmt.Sprintf("incorrect file format: %s", ext))
	}

	s := strings.Split(strings.TrimSuffix(fn, ext), "_")
	if len(s) != 2 || len(s[0]) == 0 || len(s[1]) == 0 {
		return "", "", errors.New(fmt.Sprintf("incorrect filename: %s", fn))
	}

	return s[0], s[1], nil
}

func getConfig(ctx context.Context, accessToken string, owner string, repo string, ref string) *Config {
	data, _, _ := gh.GetBlob(ctx, accessToken, owner, repo, ref, configPath)
	return parseConfig(data)
}

func parseConfig(data []byte) *Config {
	cfg := &Config{}

	err := yaml.Unmarshal(data, cfg)
	if err != nil {
		json.Unmarshal(data, cfg)
	}

	return cfg
}

func GetContentTypeEntries(cli *gontentful.Client, contenType string) (*gontentful.Entries, error) {
	var wg sync.WaitGroup
	var err error

	first, err := cli.Entries.GetEntries(createQuery(contentType, queryLimit, 0))
	if err != nil {
		return nil, err
	}

	res := &gontentful.Entries{
		Items: first.Items,
		Limit: first.Limit,
		Total: first.Total,
	}

	if first.Total > first.Limit && first.Total > 0 && first.Limit > 0 {
		rest := int(math.Floor(float64(first.Total / first.Limit)))
		if math.Mod(float64(first.Total), float64(first.Limit)) == 0 {
			rest = rest - 1
		}
		wg.Add(rest)
		items := make([][]*gontentful.Entry, rest)

		for i := 1; i <= rest; i++ {
			go func(page int) {
				defer wg.Done()
				ctnt, err := cli.Entries.GetEntries(createQuery(contentType, queryLimit, page))
				if err != nil {
					return
				}
				items[page-1] = ctnt.Items
			}(i)
		}
		wg.Wait()
		if err != nil {
			return nil, err
		}
		for _, i := range items {
			res.Items = append(res.Items, i...)
		}
		res.Total = int(len(res.Items))
	}

	return res, nil
}

func GetAllEntries(cli *gontentful.Client) (*gontentful.Entries, error) {
	res := &gontentful.Entries{}

	_, err := cli.Spaces.SyncPaged("", func(sr *gontentful.SyncResponse) error {
		for _, item := range sr.Items {
			res.Items = append(res.Items, item)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	res.Total = len(res.Items)
	return res, nil
}

func createQuery(contentType string, limit int, page int) url.Values {
	return url.Values{
		"content_type": []string{contentType},
		"limit":        []string{fmt.Sprint(limit)},
		"skip":         []string{fmt.Sprint(limit * page)},
		"locale":       []string{"*"},
		"include":      []string{"0"},
	}
}

func toCamelCase(s string) string {
	return snake.ReplaceAllStringFunc(s, func(w string) string {
		return strings.ToUpper(string(w[1]))
	})
}
