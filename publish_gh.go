package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	gh "github.com/moonwalker/moonbase/pkg/github"
)

type GHPublish struct {
	Entry      *PublishedEntry
	FolderName string
	FileName   string
	Locales    *Locales
}

func NewGHPublish(entry *PublishedEntry, fileName string, locales *Locales) *GHPublish {
	folderName := entry.Sys.ContentType.Sys.ID
	return &GHPublish{
		FolderName: folderName,
		FileName:   fileName,
		Locales:    locales,
		Entry:      entry,
	}
}

func (s *GHPublish) Exec(repo string) error {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	cd, err := TransformPublishedEntry(s.Locales, s.Entry)
	if err != nil {
		return err
	}

	// upload to github
	entries := make([]gh.BlobEntry, 0)
	for l, c := range cd {
		fileName := fmt.Sprintf("%s_%s.json", s.FileName, l)
		contentBytes, err := json.Marshal(c)
		if err != nil {
			return err
		}

		content := string(contentBytes)
		path := filepath.Join(cfg.WorkDir, s.FolderName, fileName)
		entries = append(entries, gh.BlobEntry{
			Path:    path,
			Content: &content,
		})
	}

	_, err = gh.CommitBlobs(context.Background(), cfg.Token, owner, repo, branch, entries, "feat(content): update files")
	if err != nil {
		return err
	}

	return nil
}
