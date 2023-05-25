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
	Brand      string
}

func NewGHPublish(entry *PublishedEntry, fileName string, locales *Locales, brand string) *GHPublish {
	folderName := ""
	if entry.Sys.Type == ASSET {
		folderName = ASSET_TABLE_NAME
	} else {
		folderName = entry.Sys.ContentType.Sys.ID
	}
	return &GHPublish{
		FolderName: folderName,
		FileName:   fileName,
		Locales:    locales,
		Entry:      entry,
		Brand:      brand,
	}
}

func (s *GHPublish) Exec(repo string) error {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	cd, err := TransformPublishedEntry(s.Locales, s.Entry, s.Brand)
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

	if s.Entry.Sys.Type == ASSET {
		imageURLs := getAssetImageURL(s.Entry)
		for fn, url := range imageURLs {
			// download image
			imageContent, err := downloadImage(url)
			if err != nil {
				return err
			}

			// add image to entries array
			entries = append(entries, gh.BlobEntry{
				Path:    fmt.Sprintf("%s/%s", IMAGE_FOLDER_NAME, fn),
				Content: &imageContent,
			})
		}
	}

	_, err = gh.CommitBlobs(context.Background(), cfg.Token, owner, repo, branch, entries, "feat(content): update files")
	if err != nil {
		return err
	}

	return nil
}

func getAssetImageURL(entry *PublishedEntry) map[string]string {
	imageURLs := make(map[string]string)

	file, ok := entry.Fields["file"]
	if ok {
		for loc, fc := range file {
			fileContent, ok := fc.(map[string]interface{})
			if ok {
				fileName := fileContent["fileName"].(string)
				if fileName != "" {
					url := fileContent["url"].(string)
					if url != "" {
						imageURLs[GetImageFileName(fileName, entry.Sys.ID, loc)] = fmt.Sprintf("http:%s", url)
					}
				}
			}
		}
	}

	return imageURLs
}
