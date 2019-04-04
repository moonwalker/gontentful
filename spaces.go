package gontentful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

type SpacesService service

func (s *SpacesService) Get(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathSpaces, s.client.Options.SpaceID)
	return s.client.get(path, query)
}

func (s *SpacesService) GetSpace() (*Space, error) {
	data, err := s.Get(nil)
	if err != nil {
		return nil, err
	}
	res := &Space{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *SpacesService) Create(body []byte) ([]byte, error) {
	path := pathSpacesCreate
	s.client.headers[headerContentType] = "application/vnd.contentful.management.v1+json"
	s.client.headers[headerContentfulOrganization] = s.client.Options.OrgID
	return s.client.post(path, bytes.NewBuffer(body))
}
