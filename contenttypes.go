package gontentful

import (
	"fmt"
)

type ContentTypesService service

func (s *ContentTypesService) Get() ([]byte, error) {
	path := fmt.Sprintf(pathContentTypes, s.client.Options.SpaceID)
	return s.client.get(path, nil)
}
