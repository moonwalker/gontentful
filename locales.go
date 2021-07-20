package gontentful

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type LocalesService service

func (s *LocalesService) Get(query url.Values) ([]byte, error) {
	path := fmt.Sprintf(pathLocales, s.client.Options.SpaceID)
	return s.client.get(path, query)
}

func (s *LocalesService) GetLocales() (*Locales, error) {
	data, err := s.Get(nil)
	if err != nil {
		return nil, err
	}
	res := &Locales{}
	err = json.Unmarshal(data, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
