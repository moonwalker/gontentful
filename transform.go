package gontentful

import (
	"time"

	"github.com/moonwalker/moonbase/pkg/content"
)

func TransformModel(model *ContentType) (*content.Schema, error) {
	createdAt, _ := time.Parse(time.RFC3339, model.Sys.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, model.Sys.UpdatedAt)
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
		}

		if item.DefaultValue != nil {
			for _, dv := range item.DefaultValue {
				cf.DefaultValue = dv
				break
			}
		}

		if item.Required {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "required",
				Value: true,
			})
		}

		transformField(cf, item.Type, item.LinkType, item.Validations, item.Items)

		schema.Fields = append(schema.Fields, cf)
	}

	return schema, nil
}

func transformField(cf *content.Field, fieldType string, linkType string, validations []*FieldValidation, items *FieldTypeArrayItem) {
	for _, v := range validations {
		if v.Unique {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "unique",
				Value: true,
			})
		}
		if v.In != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "in",
				Value: v.In,
			})
		}
		if v.Size != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "size",
				Value: *v.Size,
			})
		}
		if v.Range != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "range",
				Value: *v.Range,
			})
		}
		if v.Regexp != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "regexp",
				Value: *v.Regexp,
			})
		}
	}

	switch fieldType {
	case "Symbol":
		cf.Type = "text"
		break
	case "Boolean":
		cf.Type = "bool"
		break
	case "Integer":
		cf.Type = "int"
		break
	case "Number":
		cf.Type = "float"
		break
	case "Text":
		cf.Type = "longtext"
		break
	case "Link":
		cf.Reference = true
		if linkType == "Asset" {
			cf.Type = "_asset"
		} else {
			cf.Type = getFieldLinkContentType(validations)
		}
		break
	case "Array":
		cf.List = true
		transformField(cf, items.Type, items.LinkType, items.Validations, nil)
		break
	}
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
