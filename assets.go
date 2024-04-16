package gontentful

import (
	"bytes"
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	ASSET              = "Asset"
	DELETED_ASSET      = "DeletedAsset"
	ASSET_TABLE_NAME   = "_asset"
	ASSET_DISPLAYFIELD = "title"
	IMAGE_FOLDER_NAME  = "_images"
)

var (
	assetColumns          = []string{"title", "description", "file_name", "content_type", "url"}
	localizedAssetColumns = map[string]bool{
		"title":       true,
		"description": true,
		"file":        true,
	}
)

type PGSQLAssetTable struct {
	Name         string
	Columns      []string
	FieldName    string
	DisplayField string
}

func NewPGSQLAssetTable() *PGSQLAssetTable {
	return &PGSQLAssetTable{
		Name:         ASSET_TABLE_NAME,
		Columns:      assetColumns,
		FieldName:    ASSET,
		DisplayField: ASSET_DISPLAYFIELD,
	}
}

func (a *PGSQLAssetTable) Fields() []map[string]string {
	fields := make([]map[string]string, 0)
	for _, c := range assetColumns {
		fields = append(fields, map[string]string{
			"id":    c,
			"type":  "text",
			"label": cases.Title(language.Und).String(c),
		})
	}
	return fields
}

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
