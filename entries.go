package gontentful

import (
	"net/url"
	"fmt"
	"bytes"
)

type EntriesService service

func (s *EntriesService) Get(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, s.client.Options.SpaceID)
	return s.client.get(path, query)
}

func (s *EntriesService) Create(contentType string, body []byte) ([]byte, error) {
	path := fmt.Sprintf(pathEntries, s.client.Options.SpaceID)
	// Set header for content type id
	s.client.headers[headerContentfulContentType] = contentType
	return s.client.post(path, bytes.NewBuffer(body))
}
