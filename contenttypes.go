package gontentful

import (
	"fmt"
	"bytes"
)

type ContentTypesService service

func (s *ContentTypesService) Get() ([]byte, error) {
	path := fmt.Sprintf(pathContentTypes, s.client.Options.SpaceID)
	return s.client.get(path, nil)
}

func (s *ContentTypesService) GetSingle(contentTypeId string) ([]byte, error) {
	path := fmt.Sprintf(pathContentType, s.client.Options.SpaceID, contentTypeId)
	return s.client.get(path, nil)
}

func (s *ContentTypesService) Update(contentType string, body []byte, version string) ([]byte, error) {
	path := fmt.Sprintf(pathContentType, s.client.Options.SpaceID, contentType)
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, bytes.NewBuffer(body))
}

func (s *ContentTypesService) Create(contentType string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathContentType, s.client.Options.SpaceID, contentType)
	return s.client.put(path, bytes.NewBuffer(body))
}

func (s *ContentTypesService) Publish(contentType string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathContentTypesPublish, s.client.Options.SpaceID, contentType)
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, nil)
}
