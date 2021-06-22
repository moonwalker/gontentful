package gontentful

import (
	"bytes"
	"fmt"
)

const (
	ASSET         = "Asset"
	DELETED_ASSET = "DeletedAsset"
)

var (
	assetColumns   = []string{"title", "description", "file_name", "content_type", "url"}
	assetTableName = "_asset"
)

type AssetsService service

func (s *AssetsService) Create(body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathAssets, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	// Set header for content type
	s.client.headers[headerContentType] = "application/vnd.contentful.management.v1+json"
	return s.client.post(path, bytes.NewBuffer(body))
}

func (s *AssetsService) Process(id string, locale string) ([]byte, error) {
	path := fmt.Sprintf(pathAssetsProcess, s.client.Options.SpaceID, s.client.Options.EnvironmentID, id, locale)
	return s.client.put(path, nil)
}

func (s *AssetsService) Publish(id string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathAssetsPublished, s.client.Options.SpaceID, s.client.Options.EnvironmentID, id)
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, nil)
}
