package gontentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	DELETED_CONTENT_TYPE = "DeletedContentType"
)

type ContentTypesService service

func (s *ContentTypesService) Get(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathContentTypes, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	return s.client.get(path, query)
}

func (s *ContentTypesService) GetTypes() (*ContentTypes, error) {
	data, err := s.Get(nil)
	if err != nil {
		return nil, err
	}
	res := &ContentTypes{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *ContentTypesService) GetSingle(contentTypeId string) ([]byte, error) {
	path := fmt.Sprintf(pathContentType, s.client.Options.SpaceID, s.client.Options.EnvironmentID, contentTypeId)
	return s.client.get(path, nil)
}

func (s *ContentTypesService) Update(contentType string, body []byte, version string) ([]byte, error) {
	path := fmt.Sprintf(pathContentType, s.client.Options.SpaceID, s.client.Options.EnvironmentID, contentType)
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, bytes.NewBuffer(body))
}

func (s *ContentTypesService) Create(contentType string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathContentType, s.client.Options.SpaceID, s.client.Options.EnvironmentID, contentType)
	return s.client.put(path, bytes.NewBuffer(body))
}

func (s *ContentTypesService) Publish(contentType string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathContentTypesPublish, s.client.Options.SpaceID, s.client.Options.EnvironmentID, contentType)
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, nil)
}

func (s *ContentTypesService) UnPublish(contentType string) ([]byte, error) {
	path := fmt.Sprintf(pathContentTypesPublish, s.client.Options.SpaceID, s.client.Options.EnvironmentID, contentType)
	return s.client.delete(path)
}

func (s *ContentTypesService) Delete(contentType string) ([]byte, error) {
	path := fmt.Sprintf(pathContentType, s.client.Options.SpaceID, s.client.Options.EnvironmentID, contentType)
	return s.client.delete(path)
}

func (s *ContentTypesService) GetCMATypes() (*ContentTypes, error) {
	path := fmt.Sprintf(pathContentTypes, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	data, err := s.client.getCMA(path, nil)
	if err != nil {
		return nil, err
	}
	res := &ContentTypes{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
