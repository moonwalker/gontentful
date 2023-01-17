package gontentful

import (
	"github.com/moonwalker/moonbase/pkg/content"
)

func TransformModel(model *ContentType) (*content.Schema, error) {
	panic("not implemented")
}

func FormatSchema(schema *content.Schema) (*ContentType, error) {
	panic("not implemented")
}

func TransformEntry(model *Entry) (*content.ContentData, error) {
	data := &content.ContentData{
		ID:     model.Sys.ID,
		Fields: model.Fields,
	}
	data.Fields["Version"] = model.Sys.Version
	data.Fields["CreatedAt"] = model.Sys.CreatedAt
	data.Fields["UpdatedAt"] = model.Sys.UpdatedAt

	return data, nil
}

func FormatData(data *content.ContentData) (*Entry, error) {
	panic("not implemented")
}
