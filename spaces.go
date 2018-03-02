package gontentful

import (
	"net/url"
	"fmt"
)

type SpacesService service

func (s *SpacesService) Get(query url.Values, preview bool) ([]byte, error) {
	path := fmt.Sprintf(pathSpaces, s.client.Options.SpaceID)
	return s.client.get(path, query, preview)
}
