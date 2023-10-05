package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gosimple/slug"

	"github.com/moonwalker/gontentful"
)

const (
	queryLimit   = 1000
	owner        = "moonwalker"
	branch       = "main"
	configPath   = "moonbase.yaml"
	include      = 0
	outputFormat = "./_output/%s"
)

type Config struct {
	WorkDir string `json:"workdir" yaml:"workdir"`
}

func transformContent() {
	start := time.Now()
	fmt.Println("Content transforming started")

	opts := &gontentful.ClientOptions{
		SpaceID:       spaceID,
		EnvironmentID: "master",
		CdnToken:      cdnToken,
		CdnURL:        apiURL,
		CmaToken:      cmaToken,
		CmaURL:        cmaURL,
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
	if res.Total == 0 {
		log.Fatalf("entries not found: %s", contentType)
	}

	locales, err := cli.Locales.GetLocales()
	if err != nil {
		log.Fatalf("failed to fetch locales: %s", err.Error())
	}
	defaultLocale := "en"
	for _, l := range locales.Items {
		if l.Default {
			defaultLocale = l.Code
			break
		}
	}

	var ctres *gontentful.ContentTypes
	if len(contentType) > 0 {
		var ctype *gontentful.ContentType
		ctype, err = cli.ContentTypes.GetSingleCMA(contentType)
		if err == nil {
			ctres = &gontentful.ContentTypes{
				Total: 1,
				Items: []*gontentful.ContentType{ctype},
			}
		}
	} else {
		ctres, err = cli.ContentTypes.GetCMATypes()
	}
	if err != nil {
		log.Fatalf("failed to fetch contenttypes: %s", err.Error())
	}
	if ctres.Total == 0 {
		log.Fatalf("contenttype(s) not found: %s", contentType)
	}

	displayFields := make(map[string]string)
	for _, ct := range ctres.Items {
		displayFields[ct.Sys.ID] = ct.DisplayField
	}
	displayFields[gontentful.ASSET_TABLE_NAME] = gontentful.ASSET_DISPLAYFIELD

	imageURLs := make(map[string]string)

	for _, item := range res.Items {
		ct := contentType
		isAsset := item.Sys.Type == gontentful.ASSET

		if isAsset {
			ct = gontentful.ASSET_TABLE_NAME
			if !noImages {
				gontentful.GetAssetImageURL(item, imageURLs)
			}
		} else if item.Sys.Type == gontentful.ENTRY && len(ct) == 0 {
			ct = toCamelCase(item.Sys.ContentType.Sys.ID)
		}

		if !onlyImages {
			entries := gontentful.TransformEntry(locales.Items, item, brand)

			for l, e := range entries {
				b, err := json.Marshal(e)
				if err != nil {
					log.Fatalf("failed to marshal entry: %s", err.Error())
				}

				path := fmt.Sprintf(outputFormat, ct)
				err = os.MkdirAll(path, os.ModePerm)
				if err != nil {
					log.Fatalf("failed to create output folder %s: %s", path, err.Error())
				}

				dfv := getDisplayField(item, displayFields[ct], defaultLocale)
				fn := ""
				if isAsset {
					fn = fmt.Sprintf("%s-%s_%s", dfv, item.Sys.ID, l)
				} else {
					fn = fmt.Sprintf("%s_%s", dfv, l)
				}
				f := fmt.Sprintf("%s/%s.json", path, strings.ToLower(fn))
				fmt.Printf("Writing file: %s", f)
				os.WriteFile(f, b, 0644)
				fmt.Printf("\033[2K")
				fmt.Println()
				fmt.Printf("\033[1A")
			}
		}
	}
	if onlyImages {
		fmt.Println(" Transform skipped. Only downloading iumages.")
	}

	i := 1
	j := len(imageURLs)
	errors := make([]string, 0)
	images := make(map[string]bool)
	existingImages := make(map[string]bool)
	imgPath := fmt.Sprintf(outputFormat, gontentful.IMAGE_FOLDER_NAME)

	if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
		files, err := os.ReadDir(imgPath)
		if err != nil {
			log.Fatal(err)
		}
		for _, file := range files {
			images[file.Name()] = true
		}
	}

	if j > 0 {
		err = os.MkdirAll(imgPath, os.ModePerm)
		if err != nil {
			log.Fatalf("failed to create images folder: %s", err.Error())
		}

		for fn, url := range imageURLs {
			if images[fn] {
				fmt.Printf("Skipping images: %d/%d - %s", i, j, fn)
			} else {
				fmt.Printf("Dowloading images: %d/%d - %s", i, j, fn)
				err = downloadImage(url, fmt.Sprintf("%s/%s", imgPath, fn))
				if err != nil {
					errors = append(errors, err.Error())
				}
			}
			existingImages[fn] = true
			i++
			fmt.Printf("\033[2K")
			fmt.Println()
			fmt.Printf("\033[1A")
		}
	}

	for fn := range images {
		if !existingImages[fn] {
			err = os.Remove(fmt.Sprintf("%s/%s", imgPath, fn))
			if err != nil {
				errors = append(errors, err.Error())
			}
		}
	}

	fmt.Printf("Content successfully transformed in %.1fs\n", time.Since(start).Seconds())

	for _, e := range errors {
		fmt.Println(e)
	}
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
	os.WriteFile("/tmp/entries.json", b, 0644)*/

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

func getDisplayField(e *gontentful.Entry, displayField string, defaultLocale string) string {
	for k, v := range e.Fields {
		if k == displayField {
			if i, ok := v.(map[string]interface{}); ok {
				if s, ok := i[defaultLocale].(string); ok {
					if df := slug.Make(s); df != "" {
						return df
					}
				}
			}
			break
		}
	}
	return e.Sys.ID
}

func downloadImage(URL, fileName string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return fmt.Errorf("failed fetch url %s: %s", URL, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download %s: %d - %s", fileName, resp.StatusCode, resp.Status)
	}
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %s", fileName, err.Error())
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy file %s: %s", fileName, err.Error())
	}

	return nil
}
