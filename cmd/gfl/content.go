package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/moonwalker/gontentful"
	"github.com/moonwalker/moonbase/pkg/content"
)

const (
	queryLimit = 1000
	owner      = "moonwalker"
	branch     = "main"
	configPath = "moonbase.yaml"
	include    = 0
)

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
		log.Fatalf("failed to fetch entries: %s", err.Error())
	}

	locales, err := cli.Locales.GetLocales()
	if err != nil {
		log.Fatalf("failed to fetch locales: %s", err.Error())
	}

	if res.Total > 0 {
		for _, item := range res.Items {
			var entries map[string]*content.ContentData
			entries, err = gontentful.TransformEntry(locales, item)
			if err != nil {
				log.Fatal(fmt.Errorf("failed to transform entry: %s", err.Error()))
			}

			ct := contentType

			if item.Sys.Type == "Asset" {
				ct = "_asset"
			} else if item.Sys.Type == "Entry" && len(ct) == 0 {
				ct = toCamelCase(item.Sys.ContentType.Sys.ID)
			}

			for l, e := range entries {
				b, err := json.Marshal(e)
				if err != nil {
					log.Fatalf("failed to marshal entry: %s", err.Error())
				}

				path := fmt.Sprintf("./output/%s", ct)
				err = os.MkdirAll(path, os.ModePerm)
				if err != nil {
					log.Fatalf("failed to create output folder %s: %s", path, err.Error())
				}

				fn := fmt.Sprintf("%s_%s", item.Sys.ID, l)
				fmt.Printf("Writing file: %s/%s.json", path, strings.ToLower(fn))
				ioutil.WriteFile(fmt.Sprintf("%s/%s.json", path, strings.ToLower(fn)), b, 0644)
				fmt.Printf("\033[2K")
				fmt.Println()
				fmt.Printf("\033[1A")
			}
		}
	}

	fmt.Println("\r\nContent successfully transformed")
}

func formatContent() {
	fmt.Println("Content formatting started")

	entries, _, err := gontentful.GetCMSEntries(contentType, repo, include)
	if err != nil {
		log.Fatalf("failed to format file content: %s", err.Error())
	}

	fmt.Println("Entries count:", len(entries.Items))

	// tmp debug
	/*b, err := json.Marshal(*entries)
	if err != nil {
		log.Fatal(err.Error())
	}
	ioutil.WriteFile("/tmp/entries.json", b, 0644)*/

	fmt.Println("Content successfully formatted")
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
		rest := int(first.Total / first.Limit)
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
		res.Items = append(res.Items, sr.Items...)
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
