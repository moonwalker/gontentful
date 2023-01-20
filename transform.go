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
			List:      item.Type == "Array",
			Reference: item.Type == "Link",
		}

		if item.Required {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "required",
				Value: true,
			})
		}

		if item.DefaultValue != nil {
			for _, dv := range item.DefaultValue {
				cf.DefaultValue = dv
				break
			}
		}

		if item.Type == "Array" {
			transformField(cf, item.Items.Type, item.Items.LinkType, item.Items.Validations)
		} else {
			transformField(cf, item.Type, item.LinkType, item.Validations)
		}

		schema.Fields = append(schema.Fields, cf)
	}

	return schema, nil
}

func transformField(cf *content.Field, fieldType string, linkType string, validations []*FieldValidation) {
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
	}

	for _, v := range validations {
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
		if v.Size != nil {
			cv := &content.Validation{
				Type:  "size",
				Value: v.Size,
			}
			cf.Validations = append(cf.Validations, cv)
		}
		if v.Range != nil {
			cv := &content.Validation{
				Type:  "range",
				Value: v.Range,
			}
			cf.Validations = append(cf.Validations, cv)
		}
		if v.Regexp != nil {
			cv := &content.Validation{
				Type:  "regexp",
				Value: v.Regexp,
			}
			cf.Validations = append(cf.Validations, cv)
		}
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
