package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/moonwalker/gontentful"
	"github.com/spf13/cobra"
)

const queryLimit = 1000

var (
	direction,
	contentType,
	repo string
	snake = regexp.MustCompile(`([_ ]\w)`)
)

var (
	contentMigrateCmd = &cobra.Command{
		Use:   "migrateContent",
		Short: "migrate content",
		PreRun: func(cmd *cobra.Command, args []string) {
			if direction == "toContentful" {
				cmd.MarkPersistentFlagRequired("repo")
			} else {
				rootCmd.MarkPersistentFlagRequired("space")
				rootCmd.MarkPersistentFlagRequired("token")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			switch direction {
			case "toContentful":
				formatContent()
			case "fromContentful":
				transformContent()
			}
		},
	}
)

func init() {
	// type of the content to migrate
	contentMigrateCmd.Flags().StringVarP(&contentType, "contentModel", "m", "", "type of the content to migrate")
	contentMigrateCmd.Flags().StringVarP(&repo, "repo", "r", "", "repo of the content to migrate")
	contentMigrateCmd.PersistentFlags().StringVarP(&direction, "direction", "d", "", "directions: <fromContentful|toContentful>")
	contentMigrateCmd.MarkPersistentFlagRequired("direction")
	rootCmd.AddCommand(contentMigrateCmd)
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

	if res.Total > 0 {
		for _, item := range res.Items {
			if item.Sys.Type != "Entry" {
				continue
			}
			e, err := gontentful.TransformEntry(item)
			if err != nil {
				log.Fatal(errors.New(fmt.Sprintf("failed to transform entry: %s", err.Error())))
			}
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
			fmt.Println(fmt.Sprintf("Writing file: %s/%s.json", path, item.Sys.ID))
			ioutil.WriteFile(fmt.Sprintf("%s/%s.json", path, item.Sys.ID), b, 0644)
		}
	}

	fmt.Println("Content successfully transformed")
}

func formatContent() {
	fmt.Println("Content formatting started")

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
			fmt.Print(fmt.Sprintf(""))
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
		"include":      []string{"0"},
	}
}

func toCamelCase(s string) string {
	return snake.ReplaceAllStringFunc(s, func(w string) string {
		return strings.ToUpper(string(w[1]))
	})
}
