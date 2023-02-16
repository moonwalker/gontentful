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

	"github.com/google/go-github/v48/github"
	"github.com/moonwalker/gontentful"
	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
	"gopkg.in/yaml.v3"
)

const (
	queryLimit = 1000
	owner      = "moonwalker"
	branch     = ""
	configPath = "moonbase.yaml"
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

	ctx := context.Background()
	cfg := getConfig(ctx, accessToken, owner, repo, branch)

	localizedData := make(map[string]map[string]content.ContentData)

	path := filepath.Join(cfg.WorkDir, contentType)
	rcs, _, err := gh.GetContentsRecursive(ctx, accessToken, owner, repo, branch, path)
	if err != nil {
		log.Fatal(errors.New(fmt.Sprintf("failed to get json(s) from github: %s", err.Error())))
	}

	for _, rc := range rcs {
		id, loc, err := parseFileName(*rc.Name)
		if err != nil {
			fmt.Println(fmt.Sprintf("Skipping file: %s Err: %s", *rc.Path, err.Error()))
			continue
		}

		ghc, err := rc.GetContent()
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("RepositoryContent.GetContent failed: %s", err.Error())))
		}

		data := content.ContentData{}
		err = json.Unmarshal([]byte(ghc), &data)
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("failed to unmarshal file content(%s): %s", *rc.Path, err.Error())))
		}

		if localizedData[id] == nil {
			localizedData[id] = make(map[string]content.ContentData)
		}
		localizedData[id][loc] = data
	}

	entries := &gontentful.Entries{
		Sys: &gontentful.Sys{
			Type: "Array",
		},
	}
	for id, locData := range localizedData {
		entry, err := gontentful.FormatData(contentType, id, locData)
		if err != nil {
			log.Fatal(errors.New(fmt.Sprintf("failed to format file content: %s", err.Error())))
		}
		entries.Items = append(entries.Items, entry)
	}
	entries.Total = len(entries.Items)

	fmt.Println(fmt.Sprintf("%+v", *entries))
	fmt.Println("Content successfully formatted")
}

func getJsonsRecursive(ctx context.Context, owner string, repo string, branch string, path string) (resp []*github.RepositoryContent, err error) {
	var rcs []*github.RepositoryContent
	rcs, _, err = gh.GetTree(ctx, accessToken, owner, repo, branch, path)
	if err != nil {
		log.Fatal(errors.New(fmt.Sprintf("failed to get json(s) from github: %s", err.Error())))
	}

	for _, rc := range rcs {
		if *rc.Type == "dir" {
			var rcsr []*github.RepositoryContent
			rcsr, err = getJsonsRecursive(ctx, owner, repo, branch, *rc.Path)
			if err != nil {
				return
			}
			resp = append(resp, rcsr...)
		} else {
			resp = append(resp, rc)
		}
	}

	return
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
