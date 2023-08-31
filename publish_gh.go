package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v48/github"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

type GHPublish struct {
	Entry      *PublishedEntry
	FolderName string
	FileName   string
	Locales    *Locales
}

func NewGHPublish(entry *PublishedEntry, fileName string, locales *Locales) *GHPublish {
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
	}
}

func (s *GHPublish) Exec(repo string) ([]gh.BlobEntry, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	entries := make([]gh.BlobEntry, 0)
	imageEntries := make([]gh.BlobEntry, 0)
	if s.Entry.Sys.Type == ASSET {
		imageURLs := getAssetImageURL(s.Entry)
		for fn, url := range imageURLs {
			// download image
			imageContent, err := downloadImage(url)
			if err != nil {
				return nil, err
			}

			// create the blobs with the image's content (encoding base64)
			encoding := "base64"
			blob, _, err := gh.CreateBlob(ctx, cfg.Token, owner, repo, branch, &imageContent, &encoding)
			if err != nil {
				return nil, err
			}

			// add image sha to entries array
			entries = append(entries, gh.BlobEntry{
				Path: fmt.Sprintf("%s/%s", IMAGE_FOLDER_NAME, fn),
				SHA:  blob.SHA,
			})
			/*imageEntries = append(entries, gh.BlobEntry{
				Path: fmt.Sprintf("%s/%s", IMAGE_FOLDER_NAME, fn),
				SHA:  blob.SHA,
			})*/
		}
	}

	cflId := strings.TrimPrefix(repo, "cms-")
	cflId = strings.TrimPrefix(cflId, "mw-")
	cd, err := TransformPublishedEntry(s.Locales, s.Entry, cflId)
	if err != nil {
		return nil, err
	}

	// upload to github
	for l, c := range cd {
		fileName := fmt.Sprintf("%s_%s.json", s.FileName, l)
		contentBytes, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}

		content := string(contentBytes)
		path := filepath.Join(cfg.WorkDir, s.FolderName, fileName)
		entries = append(entries, gh.BlobEntry{
			Path:    path,
			Content: &content,
		})
	}

	if len(imageEntries) > 0 {
		_, err = gh.CommitBlobs(ctx, cfg.Token, owner, repo, branch, imageEntries, "feat(content): update images")
		if err != nil {
			return nil, err
		}
	}

	return entries, nil
}

func PublishCFChanges(repo string, entries []gh.BlobEntry) (github.Rate, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, repo, branch)

	resp, err := gh.CommitBlobs(ctx, cfg.Token, owner, repo, branch, entries, "feat(content): update files")
	return resp.Rate, err
}

func getAssetImageURL(entry *PublishedEntry) map[string]string {
	imageURLs := make(map[string]string)

	for loc, fc := range entry.Fields["file"] {
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

	return imageURLs
}
