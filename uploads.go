package gontentful

import (
	"fmt"
	"io"
)

type UploadsService service

func (s *UploadsService) Create(data io.Reader) ([]byte, error) {
	path := fmt.Sprintf(pathUploads, s.client.Options.SpaceID)
	// Set header for content type
	s.client.headers[headerContentType] = "application/octet-stream"

	return s.client.post(path, data)
}
