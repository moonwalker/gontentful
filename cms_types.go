package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

func GetCMSSchemas(repo string, ct string) (*ContentTypes, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, accessToken, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, ct)
	res, _, err := gh.GetSchemasRecursive(ctx, accessToken, owner, repo, branch, path)
	if err != nil {
		log.Fatal(fmt.Errorf("Failed to get schemas from github: %s", err.Error()))
	}

	schemas := &ContentTypes{
		Total: len(res),
		Limit: 0,
		Skip:  0,
		Items: make([]*ContentType, 0),
	}

	for _, rc := range res {
		ect := extractContentype(*rc.Path)

		ghc, err := rc.GetContent()
		if err != nil {
			log.Fatal(fmt.Errorf("RepositoryContent.GetContent failed: %s", err.Error()))
		}
		m := &content.Schema{}
		_ = json.Unmarshal([]byte(ghc), m)
		if err != nil {
			log.Fatal(fmt.Errorf("Failed to unmarshal schema %s: %s", ect, err.Error()))
		}
		t, err := FormatSchema(m)
		if err != nil {
			log.Fatal(fmt.Errorf("Failed to format schema: %s", err.Error()))
		}
		schemas.Items = append(schemas.Items, t)
	}
	return schemas, nil
}
