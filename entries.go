package gontentful

import (
	"net/url"
	"fmt"
	"bytes"
)

type EntriesService service

func (s *EntriesService) Get(query url.Values, preview bool) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, s.client.Options.SpaceID)
	return s.client.get(path, query, preview)
}

func (s *EntriesService) Create(contentType string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, s.client.Options.SpaceID)
	// Set header for content type
	s.client.headers[headerContentfulContentType] = contentType
	return s.client.post(path, bytes.NewBuffer(body))
}

func (s *EntriesService) Update(version string, entryId string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathEntry, s.client.Options.SpaceID, entryId)
	// Set header for content type
	s.client.headers[headerContentulVersion] = version
	return s.client.put(path, bytes.NewBuffer(body))
}

func (s *EntriesService) Publish(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesPublish, s.client.Options.SpaceID, entryId)
	// Set header for version
	s.client.headers[headerContentulVersion] = version
	return s.client.put(path, nil)
}

func (s *EntriesService) UnPublish(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesPublish, s.client.Options.SpaceID, entryId)
	// Set header for version
	s.client.headers[headerContentulVersion] = version
	return s.client.delete(path)
}

func (s *EntriesService) Delete(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntry, s.client.Options.SpaceID, entryId)
	// Set header for version
	s.client.headers[headerContentulVersion] = version
	return s.client.delete(path)
}

func (s *EntriesService) Archive(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesArchive, s.client.Options.SpaceID, entryId)
	// Set header for version
	s.client.headers[headerContentulVersion] = version
	return s.client.put(path, nil)
}

func (s *EntriesService) UnArchive(entryId string, version string) ([]byte, error) {
	path := fmt.Sprintf(pathEntriesArchive, s.client.Options.SpaceID, entryId)
	// Set header for version
	s.client.headers[headerContentulVersion] = version
	return s.client.delete(path)
}
