package gontentful

import (
	"context"
	"fmt"
	"path/filepath"

	gh "github.com/moonwalker/moonbase/pkg/github"
)

type GHDeleteContentType struct {
	FolderName string
	SysID      string
}

func NewGHDeleteContentType(sys *Sys) *GHDeleteContentType {
	folderName := sys.ID
	return &GHDeleteContentType{
		FolderName: folderName,
		SysID:      sys.ID,
	}
}

func (s *GHDeleteContentType) Exec(repo string) error {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	path := filepath.Join(cfg.WorkDir, s.FolderName)
	fileNames := make([]string, 0)
	fileNames = append(fileNames, fmt.Sprintf("%s/_schema.json", path))

	_, err := gh.DeleteFiles(ctx, cfg.Token, owner, repo, branch, path, "feat(content): delete files", fileNames)
	return err
}
