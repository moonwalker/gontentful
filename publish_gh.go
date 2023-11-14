package gontentful

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/google/go-github/v48/github"
	gh "github.com/moonwalker/moonbase/pkg/github"
)

type GHPublish struct {
	Entry           *PublishedEntry
	RepoName        string
	FileName        string
	Locales         []*Locale
	LocalizedFields map[string]bool
}

func NewGHPublish(entry *PublishedEntry, repoName, fileName string, locales *Locales, localizedFields map[string]bool) *GHPublish {
	return &GHPublish{
		Entry:           entry,
		RepoName:        repoName,
		FileName:        fileName,
		Locales:         locales.Items,
		LocalizedFields: localizedFields,
	}
}

func getDeleteEntries(ctx context.Context, cfg *Config, s *GHPublish, folderName string) ([]gh.BlobEntry, error) {
	path := filepath.Join(cfg.WorkDir, folderName, s.FileName)

	fileNames := make([]string, 0)

	for _, l := range s.Locales {
		fileNames = append(fileNames, fmt.Sprintf("%s.json", l.Code))
	}

	return gh.GetDeleteFileEntries(ctx, cfg.Token, owner, s.RepoName, branch, path, "feat(content): delete files", fileNames)
}

func getAssetImages(ctx context.Context, cfg *Config, repo, url string) (*string, error) {
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
	return blob.SHA, nil
}

func (s *GHPublish) Exec() ([]gh.BlobEntry, error) {
	ctx := context.Background()
	cfg := getConfig(ctx, owner, s.RepoName, branch)

	entryType := s.Entry.Sys.Type
	entries := make([]gh.BlobEntry, 0)
	folderName := ""

	switch entryType {
	case DELETED_ASSET:
		return getDeleteEntries(ctx, cfg, s, ASSET_TABLE_NAME)
	case DELETED_ENTRY:
		return getDeleteEntries(ctx, cfg, s, s.Entry.Sys.ContentType.Sys.ID)
	case ASSET:
		folderName = ASSET_TABLE_NAME
		imageURLs := getAssetImageURL(s.Entry)
		for fn, url := range imageURLs {
			sha, err := getAssetImages(ctx, cfg, s.RepoName, url)
			if err != nil {
				return nil, err
			}
			// add image sha to entries array
			entries = append(entries, gh.BlobEntry{
				Path: fmt.Sprintf("%s/%s", IMAGE_FOLDER_NAME, fn),
				SHA:  sha,
			})
		}
	default:
		folderName = s.Entry.Sys.ContentType.Sys.ID
	}

	cflId := GetCloudflareImagesID(s.RepoName)
	cd := TransformPublishedEntry(s.Locales, s.Entry, s.LocalizedFields, cflId)

	// upload to github
	for l, c := range cd {
		fileName := fmt.Sprintf("%s/%s.json", s.FileName, l)
		contentBytes, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}

		content := string(contentBytes)
		path := filepath.Join(cfg.WorkDir, folderName, fileName)
		entries = append(entries, gh.BlobEntry{
			Path:    path,
			Content: &content,
		})
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
