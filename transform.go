package gontentful

import (
	"fmt"
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
		// if v.Unique {
		cf.Validations = append(cf.Validations, &content.Validation{
			Type:  "unique",
			Value: v.Unique,
		})
		// }
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

func transformToContentfulField(cf *ContentTypeField, fieldType string, validations []*content.Validation, list bool, reference bool) {
	for _, v := range validations {
		if v.Type == "unique" && v.Value == true {
			cf.Validations = append(cf.Validations, &FieldValidation{
				Unique: true,
			})
		}
		if v.Type == "required" && v.Value == true {
			cf.Required = true
		} else {
			cf.Required = false
		}
		if v.Type == "in" {
			fmt.Println("Validations has In value type.", v.Value)
			strarr := make([]string, 0)
			for _, i := range v.Value.([]interface{}) {
				strarr = append(strarr, i.(string))
			}
			cf.Validations = append(cf.Validations, &FieldValidation{
				In: strarr,
			})
		}
		if v.Type == "size" {
			m, _ := v.Value.(map[string]interface{})
			rv := &RangeValidation{}
			for k, v := range m {
				if k == "min" {
					rv.Min = v.(*int)
				}
				if k == "max" {
					rv.Max = v.(*int)
				}
			}
			cf.Validations = append(cf.Validations, &FieldValidation{
				Size: rv,
			})
		}
		if v.Type == "regexp" {
			m, _ := v.Value.(map[string]interface{})
			rv := &RegexpValidation{}
			for k, v := range m {
				if k == "pattern" {
					rv.Pattern = v.(int)
				}
				if k == "flags" {
					rv.Flags = v.(int)
				}
			}
			cf.Validations = append(cf.Validations, &FieldValidation{
				Regexp: rv,
			})
		}
	}

	switch fieldType {
	case "text":
		cf.Type = "Symbol"
		break
	case "bool":
		cf.Type = "Boolean"
		break
	case "int":
		cf.Type = "Integer"
		break
	case "float":
		cf.Type = "Number"
		break
	case "longtext":
		cf.Type = "Text"
		break
	case "_asset":
		cf.LinkType = "Asset"
	}

	if list {
		fmt.Println("Is List:", fieldType)
		cf.Type = "Array"
		cf.Items = &FieldTypeArrayItem{
			Type:     "Link",
			LinkType: "Entry",
		}
	}
	if reference {
		cf.Type = "Link"
		cf.Validations = append(cf.Validations, &FieldValidation{
			LinkContentType: append(make([]string, 0), fieldType),
		})
	}
}

func FormatSchema(schema *content.Schema) (*ContentType, error) {
	ct := &ContentType{
		Name:         schema.Name,
		Description:  schema.Description,
		DisplayField: schema.Fields[0].ID,
	}

	for _, f := range schema.Fields {
		ctf := &ContentTypeField{
			ID:        f.ID,
			Name:      f.Label,
			Localized: f.Localized,
			Disabled:  f.Disabled,
		}
		if f.DefaultValue != nil {
			ctf.DefaultValue = make(map[string]interface{})
			ctf.DefaultValue["en"] = f.DefaultValue
		}
		transformToContentfulField(ctf, f.Type, f.Validations, f.List, f.Reference)

		ct.Fields = append(ct.Fields, ctf)
	}

	ct.Sys = &Sys{
		ID:        schema.ID,
		Version:   schema.Version,
		CreatedAt: schema.CreatedAt.String(),
		UpdatedAt: schema.UpdatedAt.String(),
	}
	ct.Sys.CreatedBy = &Entry{
		Sys: &Sys{
			ID: schema.CreatedBy,
		},
	}
	ct.Sys.UpdatedBy = &Entry{
		Sys: &Sys{
			ID: schema.CreatedBy,
		},
	}

	return ct, nil
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
