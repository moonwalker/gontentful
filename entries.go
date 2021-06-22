package gontentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	ENTRY         = "Entry"
	DELETED_ENTRY = "DeletedEntry"
)

type EntriesService service

func (s *EntriesService) Get(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	return s.client.get(path, query)
}

func (s *EntriesService) GetEntries(query url.Values) (*Entries, error) {
	data, err := s.Get(query)
	if err != nil {
		return nil, err
	}
	res := &Entries{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *EntriesService) GetSingle(entryId string) ([]byte, error) {
	path := fmt.Sprintf(pathEntry, s.client.Options.SpaceID, s.client.Options.EnvironmentID, entryId)
	return s.client.get(path, nil)
}

func (s *EntriesService) Create(contentType string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, s.client.Options.SpaceID, s.client.Options.EnvironmentID)
	// Set header for content type
	s.client.headers[headerContentfulContentType] = contentType
	return s.client.post(path, bytes.NewBuffer(body))
}

func (s *EntriesService) Update(version string, entryId string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathEntry, s.client.Options.SpaceID, s.client.Options.EnvironmentID, entryId)
	// Set header for content type
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, bytes.NewBuffer(body))
}

func (s *EntriesService) Publish(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesPublish, s.client.Options.SpaceID, s.client.Options.EnvironmentID, entryId)
	// Set header for version
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, nil)
}

func (s *EntriesService) UnPublish(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesPublish, s.client.Options.SpaceID, s.client.Options.EnvironmentID, entryId)
	// Set header for version
	s.client.headers[headerContentfulVersion] = version
	return s.client.delete(path)
}

func (s *EntriesService) Delete(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntry, s.client.Options.SpaceID, s.client.Options.EnvironmentID, entryId)
	// Set header for version
	s.client.headers[headerContentfulVersion] = version
	return s.client.delete(path)
}

func (s *EntriesService) Archive(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesArchive, s.client.Options.SpaceID, s.client.Options.EnvironmentID, entryId)
	// Set header for version
	s.client.headers[headerContentfulVersion] = version
	return s.client.put(path, nil)
}

func (s *EntriesService) UnArchive(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesArchive, s.client.Options.SpaceID, s.client.Options.EnvironmentID, entryId)
	// Set header for version
	s.client.headers[headerContentfulVersion] = version
	return s.client.delete(path)
}
