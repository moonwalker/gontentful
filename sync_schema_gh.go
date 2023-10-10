package gontentful

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/moonwalker/moonbase/pkg/content"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

type GHSyncSchema struct {
	FolderName string
	Schema     *ContentType
}

func NewGHSyncSchema(sys *Sys, schema *ContentType) *GHSyncSchema {
	folderName := schema.Sys.ID
	return &GHSyncSchema{
		FolderName: folderName,
		Schema:     schema,
	}
}

func (s *GHSyncSchema) Exec(repo string) error {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	ct, err := TransformModel(s.Schema)
	if err != nil {
		return err
	}
	cb, err := json.Marshal(ct)
	if err != nil {
		return err
	}
	sc := string(cb)
	path := filepath.Join(cfg.WorkDir, s.FolderName, content.JsonSchemaName)

	// Upload to github
	_, err = gh.CommitBlob(ctx, cfg.Token, owner, repo, branch, path, &sc, "feat(content type): update files")

	if err != nil {
		return err
	}
	return nil
}
