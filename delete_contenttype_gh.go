package gontentful

import (
	"context"
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

	_, err := gh.DeleteFolder(ctx, cfg.Token, owner, repo, branch, path, "feat(content): delete content type")
	return err
}
