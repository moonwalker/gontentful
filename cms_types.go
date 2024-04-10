package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

func GetSchema(ctx context.Context, cfg *Config, owner, repo, ref, contentType string) (*content.Schema, error) {
	path := filepath.Join(cfg.WorkDir, contentType)

	rc, _, err := gh.GetSchema(ctx, cfg.Token, owner, repo, ref, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema of %s: %s", contentType, err.Error())
	}
	schemaContent, err := rc.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get schema content: %s", err.Error())
	}
	schema := &content.Schema{}
	err = json.Unmarshal([]byte(schemaContent), schema)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema %s: %s", contentType, err.Error())
	}
	return schema, nil
}

func GetCMSSchema(repo string, ct string) (*ContentType, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, ct)
	res, _, err := gh.GetSchema(ctx, cfg.Token, owner, repo, branch, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema from github: %s", err.Error())
	}

	ghc, err := res.GetContent()
	if err != nil {
		return nil, fmt.Errorf("repositoryContent.GetContent failed: %s", err.Error())
	}
	m := &content.Schema{}
	_ = json.Unmarshal([]byte(ghc), m)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema %s: %s", *res.Path, err.Error())
	}

	return formatSchema(m), nil
}

func GetCMSSchemas(repo string, ct string) (*ContentTypes, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, ct)
	res, _, err := gh.GetSchemasRecursive(ctx, cfg.Token, owner, repo, branch, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas from github: %s", err.Error())
	}

	schemas := &ContentTypes{
		Total: len(res),
		Limit: 0,
		Skip:  0,
		Items: make([]*ContentType, 0),
	}

	for _, rc := range res {
		ghc, err := rc.GetContent()
		if err != nil {
			return nil, fmt.Errorf("repositoryContent.GetContent failed: %s", err.Error())
		}
		m := &content.Schema{}
		_ = json.Unmarshal([]byte(ghc), m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema %s: %s", *rc.Path, err.Error())
		}
		schemas.Items = append(schemas.Items, formatSchema(m))
	}
	return schemas, nil
}

func GetCMSSchemasExpanded(repo string, ct string) (*ContentTypes, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, ct)
	res, _, err := gh.GetSchemasRecursive(ctx, cfg.Token, owner, repo, branch, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas from github: %s", err.Error())
	}

	schemas := &ContentTypes{
		Total: len(res),
		Limit: 0,
		Skip:  0,
		Items: make([]*ContentType, 0),
	}

	for _, rc := range res {
		ghc, err := rc.GetContent()
		if err != nil {
			return nil, fmt.Errorf("repositoryContent.GetContent failed: %s", err.Error())
		}
		m := &content.Schema{}
		_ = json.Unmarshal([]byte(ghc), m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema %s: %s", *rc.Path, err.Error())
		}
		schemas.Items = append(schemas.Items, formatSchemaRecursive(m)...)
	}
	return schemas, nil
}
