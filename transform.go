package gontentful

import (
	"time"

	"github.com/moonwalker/moonbase/pkg/content"
)

func TransformModel(model *ContentType) (*content.Schema, error) {
	createdAt, _ := time.Parse("2014-11-12T11:45:26.371Z", model.Sys.CreatedAt)
	updatedAt, _ := time.Parse("2015-10-11T10:35:26.341Z", model.Sys.UpdatedAt)
	schema := &content.Schema{
		ID:          model.Sys.ID,
		Name:        model.Name,
		Description: model.Description,
		CreatedAt:   &createdAt,
		CreatedBy:   "admin@moonwalker.tech",
		UpdatedAt:   &updatedAt,
		UpdatedBy:   "admin@moonwalker.tech",
		Version:     model.Sys.Version,
	}

	for _, item := range model.Fields {
		cf := &content.Field{
			ID:        item.ID,
			Label:     item.Name,
			Localized: item.Localized,
			Disabled:  item.Disabled,
			List:      item.Type == "Array",
			Reference: item.Type == "Link",
		}
		if item.Type == "Array" {
			if item.Items.Type == "Link" {
				cf.Reference = true
				if len(item.Items.Validations) > 0 && len(item.Items.Validations[0].LinkContentType) > 0 {
					cf.Type = item.Items.Validations[0].LinkContentType[0]
				}
			}
			if item.Items.Type == "Symbol" {
				cf.Type = "text"
				if len(item.Items.Validations) > 0 {
					cv := &content.Validation{
						Type:  "in",
						Value: item.Items.Validations[0].In,
					}
					cf.Validations = append(cf.Validations, cv)
				}
			}
		} else {
			cf.Type = transformType(item)
		}
		if item.Required {
			cv := &content.Validation{
				Type:  "required",
				Value: true,
			}
			cf.Validations = append(cf.Validations, cv)
		}

		for _, v := range item.Validations {
			if v.Unique {
				cv := &content.Validation{
					Type:  "unique",
					Value: true,
				}
				cf.Validations = append(cf.Validations, cv)
			}
			if v.In != nil {
				cv := &content.Validation{
					Type:  "in",
					Value: v.In,
				}
				cf.Validations = append(cf.Validations, cv)
			}
			if v.Size != nil || v.Range != nil {
				if v.In != nil {
					cv := &content.Validation{
						Type:  "size",
						Value: v.In,
					}
					cf.Validations = append(cf.Validations, cv)
				}
				if v.Size.Min != nil {
					cv := &content.Validation{
						Type:  "min",
						Value: v.Size.Min,
					}
					cf.Validations = append(cf.Validations, cv)
				}
				if v.Size.Max != nil {
					cv := &content.Validation{
						Type:  "max",
						Value: v.Size.Max,
					}
					cf.Validations = append(cf.Validations, cv)
				}
			}
			if v.Regexp != nil {
				cv := &content.Validation{
					Type:  "regexp",
					Value: v.Regexp,
				}
				cf.Validations = append(cf.Validations, cv)
			}
		}

		schema.Fields = append(schema.Fields, cf)
	}

	return schema, nil
}

func transformType(i *ContentTypeField) string {
	returnType := ""
	switch i.Type {
	case "Symbol":
		returnType = "text"
	case "Boolean":
		returnType = "bool"
	case "Integer":
		returnType = "int"
	case "Number":
		returnType = "float"
	case "Text":
		returnType = "longtext"
	case "Link":
		if i.LinkType == "Asset" {
			returnType = "_asset"
		} else {
			if len(i.Validations) > 0 {
				returnType = i.Validations[0].LinkContentType[0]
			} else {
				returnType = i.Items.Validations[0].LinkContentType[0]
			}
		}
	}
	return returnType
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
