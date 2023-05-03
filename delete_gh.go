package gontentful

import (
	"context"
	"fmt"
	"path/filepath"

	gh "github.com/moonwalker/moonbase/pkg/github"
)

type GHDelete struct {
	FolderName string
	SysID      string
	Locales    []*Locale
}

func NewGHDelete(sys *Sys, locales *Locales) *GHDelete {
	folderName := ""
	if sys.Type == DELETED_ENTRY {
		folderName = sys.ContentType.Sys.ID
	} else if sys.Type == DELETED_ASSET {
		folderName = ASSET_TABLE_NAME
	}
	return &GHDelete{
		FolderName: folderName,
		SysID:      sys.ID,
		Locales:    locales.Items,
	}
}

func (s *GHDelete) Exec(repo string) error {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)
	path := filepath.Join(cfg.WorkDir, s.FolderName)

	fileNames := make([]string, 0)

	for _, l := range s.Locales {
		fileNames = append(fileNames, fmt.Sprintf("%s_%s.json", s.SysID, l.Code))
	}

	_, err := gh.DeleteFiles(ctx, cfg.Token, owner, repo, branch, path, "feat(content): delete files", fileNames)
	return err
}
