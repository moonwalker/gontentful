package gontentful

import (
	"bytes"
	"fmt"
	"net/url"
)

type SpacesService service

func (s *SpacesService) Get(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathSpaces, s.client.Options.SpaceID)
	return s.client.get(path, query)
}

func (s *SpacesService) Create(body []byte) ([]byte, error) {
	path := pathSpacesCreate
	s.client.headers[headerContentType] = "application/vnd.contentful.management.v1+json"
	s.client.headers[headerContentfulOrganization] = s.client.Options.OrgID
	return s.client.post(path, bytes.NewBuffer(body))
}
