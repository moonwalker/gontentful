package gontentful

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/moonwalker/moonbase/pkg/content"
)

const (
	cdnClientID = "yeGKJew8TyopStA61YrS4A"
	imageCDNFmt = "//imagedelivery.net/%s/%s/%s/public"
)

func TransformModel(model *ContentType) *content.Schema {
	createdAt, _ := time.Parse(time.RFC3339Nano, model.Sys.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, model.Sys.UpdatedAt)

	schema := &content.Schema{
		ID:           model.Sys.ID,
		Name:         model.Name,
		DisplayField: model.DisplayField,
		Description:  model.Description,
		Version:      model.Sys.Version,
		CreatedAt:    &createdAt,
		CreatedBy:    "admin@moonwalker.tech",
		UpdatedAt:    &updatedAt,
		UpdatedBy:    "admin@moonwalker.tech",
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

	return schema
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
	case "Boolean":
		cf.Type = "bool"
	case "Integer":
		cf.Type = "int"
	case "Number":
		cf.Type = "float"
	case "Text":
		cf.Type = "longtext"
	case "Link":
		cf.Reference = true
		if linkType == ASSET {
			cf.Type = ASSET_TABLE_NAME
		} else {
			cf.Type = getFieldLinkContentType(validations)
		}
	case "Array":
		cf.List = true
		transformField(cf, items.Type, items.LinkType, items.Validations, nil)
	case "Object":
		cf.Type = "json"
	}
}

func transformValidationFields(vals []*content.Validation) ([]*FieldValidation, bool) {
	cfValidations := make([]*FieldValidation, 0)
	required := false
	for _, v := range vals {
		if v.Type == "unique" && v.Value == true {
			cfValidations = append(cfValidations, &FieldValidation{
				Unique: true,
			})
		}

		if v.Type == "required" && v.Value == true {
			required = true
		}

		if v.Type == "in" {
			strarr := make([]string, 0)
			for _, i := range v.Value.([]interface{}) {
				strarr = append(strarr, i.(string))
			}
			cfValidations = append(cfValidations, &FieldValidation{
				In: strarr,
			})
		}

		if v.Type == "size" {
			m, _ := v.Value.(map[string]interface{})
			rv := &RangeValidation{}
			for k, v := range m {
				if k == "min" {
					i := int(v.(float64))
					rv.Min = &i
				}
				if k == "max" {
					i := int(v.(float64))
					rv.Max = &i
				}
			}
			cfValidations = append(cfValidations, &FieldValidation{
				Size: rv,
			})
		}

		if v.Type == "regexp" {
			m, _ := v.Value.(map[string]interface{})
			rv := &RegexpValidation{}
			for k, v := range m {
				if k == "pattern" {
					rv.Pattern = int(v.(float64))
				}
				if k == "flags" {
					rv.Flags = int(v.(float64))
				}
			}
			cfValidations = append(cfValidations, &FieldValidation{
				Regexp: rv,
			})
		}
	}
	return cfValidations, required
}

func transformToContentfulField(cf *ContentTypeField, fieldType string, validations []*content.Validation, list bool, reference bool) {

	cfVals, required := transformValidationFields(validations)
	if required {
		cf.Required = true
	}
	if list {
		cf.Type = ARRAY
		cf.Items = &FieldTypeArrayItem{}

		if reference {
			cf.Items.Type, cf.Items.LinkType, cf.Items.Validations = setReferenceField(cfVals, fieldType)
		} else {
			cf.Items.Type = GetContentfulType(fieldType)
		}
	} else if reference {
		cf.Type, cf.LinkType, cf.Validations = setReferenceField(cfVals, fieldType)
	} else {
		cf.Type = GetContentfulType(fieldType)
		cf.Validations = cfVals
	}
}

func setReferenceField(cfVals []*FieldValidation, fieldType string) (t string, lt string, vs []*FieldValidation) {
	t = LINK
	lt = ENTRY
	if fieldType == ASSET_TABLE_NAME {
		lt = ASSET
	}
	vs = append(cfVals, &FieldValidation{
		LinkContentType: append(make([]string, 0), fieldType),
	})
	return
}

func GetContentfulType(fieldType string) string {
	returnVal := ""
	switch fieldType {
	case "text":
		returnVal = "Symbol"
	case "bool":
		returnVal = "Boolean"
	case "int":
		returnVal = "Integer"
	case "float":
		returnVal = "Number"
	case "longtext":
		returnVal = "Text"
	case "_asset":
		returnVal = "Asset"
	case "json":
		returnVal = "Object"
	}
	return returnVal
}

func formatSchema(schema *content.Schema) *ContentType {
	ct := &ContentType{
		Name:         schema.Name,
		Description:  schema.Description,
		DisplayField: schema.DisplayField,
		Sys: &Sys{
			ID:      schema.ID,
			Version: schema.Version,
		},
	}
	if schema.CreatedAt != nil {
		ct.Sys.CreatedAt = schema.CreatedAt.Format(time.RFC3339Nano)
	}
	if schema.UpdatedAt != nil {
		ct.Sys.UpdatedAt = schema.UpdatedAt.Format(time.RFC3339Nano)
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

	for _, f := range schema.Fields {
		ctf := &ContentTypeField{
			ID:          f.ID,
			Name:        f.Label,
			Localized:   f.Localized,
			Disabled:    f.Disabled,
			Required:    false,
			Omitted:     false,
			Validations: make([]*FieldValidation, 0),
		}
		if f.DefaultValue != nil {
			ctf.DefaultValue = make(map[string]interface{})
			ctf.DefaultValue["en"] = f.DefaultValue
		}
		transformToContentfulField(ctf, f.Type, f.Validations, f.List, f.Reference)

		ct.Fields = append(ct.Fields, ctf)
	}

	return ct
}

func formatSchemaRecursive(schema *content.Schema) []*ContentType {
	res := make([]*ContentType, 0)

	ct := &ContentType{
		Name:         schema.Name,
		Description:  schema.Description,
		DisplayField: schema.DisplayField,
		Sys: &Sys{
			ID:      schema.ID,
			Version: schema.Version,
		},
	}
	if schema.CreatedAt != nil {
		ct.Sys.CreatedAt = schema.CreatedAt.Format(time.RFC3339Nano)
	}
	if schema.UpdatedAt != nil {
		ct.Sys.UpdatedAt = schema.UpdatedAt.Format(time.RFC3339Nano)
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

	for _, f := range schema.Fields {
		ctf := &ContentTypeField{
			ID:          f.ID,
			Name:        f.Label,
			Localized:   f.Localized,
			Disabled:    f.Disabled,
			Required:    false,
			Omitted:     false,
			Validations: make([]*FieldValidation, 0),
		}
		if f.DefaultValue != nil {
			ctf.DefaultValue = make(map[string]interface{})
			ctf.DefaultValue["en"] = f.DefaultValue
		}
		hasSchema := (f.Schema != nil)
		isReference := f.Reference || hasSchema
		fType := f.Type
		if hasSchema {
			fType = f.Schema.ID
		}

		transformToContentfulField(ctf, fType, f.Validations, f.List, isReference)

		ct.Fields = append(ct.Fields, ctf)

		if f.Schema != nil {
			res = append(res, formatSchemaRecursive(f.Schema)...)
		}
	}

	res = append(res, ct)

	return res
}

func TransformEntry(locales []*Locale, model *Entry, brand string, fmtVideoURL func(string) string) map[string]*content.ContentData {
	res := make(map[string]*content.ContentData, 0)
	for _, loc := range locales {
		data := &content.ContentData{
			ID:     model.Sys.ID,
			Fields: make(map[string]interface{}),
		}

		for fn, fv := range model.Fields {
			contentLoc := loc.Code
			locValues, ok := fv.(map[string]interface{})
			if !ok {
				continue // no locale value, continue
			}

			locValue := locValues[strings.ToLower(loc.Code)]
			if locValue == nil {
				locValue = locValues[DefaultLocale]
				contentLoc = DefaultLocale
			}

			if lsysl, ok := locValue.([]interface{}); ok {
				for _, lsyso := range lsysl {
					if lsys, ok := lsyso.(map[string]interface{}); ok {
						sid := getSysID(lsys)
						if sid == nil {
							break
						}
						if data.Fields[fn] == nil {
							data.Fields[fn] = make([]interface{}, 0)
						}
						data.Fields[fn] = append(data.Fields[fn].([]interface{}), sid)
					}
				}
			} else {
				if lsys, ok := locValue.(map[string]interface{}); ok {
					data.Fields[fn] = getSysID(lsys)
				}
			}

			if data.Fields[fn] == nil {
				if model.Sys.Type == ASSET && fn == "file" {
					data.Fields[fn] = replaceAssetFile(brand, locValue, model.Sys.ID, strings.ToLower(contentLoc), fmtVideoURL)
				} else {
					data.Fields[fn] = locValue
				}
			}
		}

		data.CreatedAt = model.Sys.CreatedAt
		data.CreatedBy = "admin"
		data.UpdatedAt = model.Sys.UpdatedAt
		data.UpdatedBy = "admin"
		data.Version = model.Sys.Version
		res[strings.ToLower(loc.Code)] = data
	}

	return res
}

func TransformPublishedEntry(locales []*Locale, model *PublishedEntry, localizedFields map[string]bool, brand string, fmtVideoURL func(string) string) map[string]*content.ContentData {
	res := make(map[string]*content.ContentData, 0)
	for _, loc := range locales {
		data := &content.ContentData{
			ID:     model.Sys.ID,
			Fields: make(map[string]interface{}),
		}

		for fn, fv := range model.Fields {
			contentLoc := loc.Code
			locValues := fv
			locValue := locValues[strings.ToLower(loc.Code)]
			if (model.Sys.Type == ASSET && !localizedAssetColumns[fn]) ||
				(model.Sys.Type != ASSET && !localizedFields[fn]) ||
				locValue == nil {
				locValue = locValues[DefaultLocale]
				contentLoc = DefaultLocale
			}

			if lsysl, ok := locValue.([]interface{}); ok {
				for _, lsyso := range lsysl {
					if lsys, ok := lsyso.(map[string]interface{}); ok {
						sid := getSysID(lsys)
						if sid == nil {
							break
						}
						if data.Fields[fn] == nil {
							data.Fields[fn] = make([]interface{}, 0)
						}
						data.Fields[fn] = append(data.Fields[fn].([]interface{}), sid)
					}
				}
			} else {
				if lsys, ok := locValue.(map[string]interface{}); ok {
					data.Fields[fn] = getSysID(lsys)
				}
			}

			if data.Fields[fn] == nil {
				if model.Sys.Type == ASSET && fn == "file" {
					data.Fields[fn] = replaceAssetFile(brand, locValue, model.Sys.ID, strings.ToLower(contentLoc), fmtVideoURL)
				} else {
					data.Fields[fn] = locValue
				}
			}
		}

		data.CreatedAt = model.Sys.CreatedAt
		data.CreatedBy = "admin"
		data.UpdatedAt = model.Sys.UpdatedAt
		data.UpdatedBy = "admin"
		data.PublishedAt = model.Sys.PublishedAt
		data.PublishedBy = "admin"
		data.Status = model.Sys.Status()
		data.Version = model.Sys.Version
		res[strings.ToLower(loc.Code)] = data
	}

	return res
}

func getSysDate(date string) *time.Time {
	var t time.Time
	t, _ = time.Parse(time.RFC3339Nano, date)
	return &t
}

func getSysID(lsys map[string]interface{}) interface{} {
	if sys, ok := lsys["sys"].(map[string]interface{}); ok {
		if sid, ok := sys["id"].(string); ok {
			return sid
		}
	}
	return nil
}

func FormatData(contentType string, id string, schemas map[string]*content.Schema, locData map[string]map[string]map[string]content.ContentData) (*Entry, map[string]string, error) {
	schema := schemas[contentType]
	contents := locData[contentType][id]

	if schema == nil {
		return nil, nil, fmt.Errorf("missing schema: %s", contentType)
	}
	if contents == nil {
		return nil, nil, fmt.Errorf("missing content: %s %s", contentType, id)
	}

	refFields := make(map[string]*content.Field, 0)
	for _, sf := range schema.Fields {
		if sf.Reference {
			refFields[sf.ID] = sf
		}
	}

	entry, includes := formatEntry(id, contentType, contents, refFields)

	return entry, includes, nil
}

func formatEntry(id string, contentType string, contents map[string]content.ContentData, refFields map[string]*content.Field) (*Entry, map[string]string) {
	includes := make(map[string]string)

	sysType := ENTRY
	if contentType == ASSET_TABLE_NAME {
		sysType = ASSET
	}

	e := &Entry{
		Sys: &Sys{
			ID:      id,
			Type:    sysType,
			Version: contents[DefaultLocale].Version,
			ContentType: &ContentType{
				Sys: &Sys{
					Type:     LINK,
					LinkType: CONTENT_TYPE,
					ID:       contentType,
				},
			},
		},
	}

	e.Sys.CreatedAt = contents[DefaultLocale].CreatedAt
	e.Sys.UpdatedAt = contents[DefaultLocale].UpdatedAt

	fields := make(map[string]interface{})

	for loc, data := range contents {
		for fn, fv := range data.Fields {
			if fv == nil {
				continue
			}
			if fields[fn] == nil {
				fields[fn] = make(map[string]interface{})
			}

			if rf := refFields[fn]; rf != nil {
				if rf.List {
					if rl, ok := fv.([]interface{}); ok {
						refList := make([]interface{}, 0)
						for _, r := range rl {
							if rid, ok := r.(string); ok {
								esys := make(map[string]interface{})
								esys["type"] = LINK
								esys["linkType"] = ENTRY
								esys["id"] = rid
								es := make(map[string]interface{})
								es["sys"] = esys
								refList = append(refList, es)
								includes[rid] = rf.Type
							}
						}
						fields[fn].(map[string]interface{})[loc] = refList
					}
				} else {
					esys := make(map[string]interface{})
					esys["type"] = LINK
					esys["linkType"] = ENTRY
					esys["id"] = fv
					es := make(map[string]interface{})
					es["sys"] = esys
					fields[fn].(map[string]interface{})[loc] = es
					includes[fv.(string)] = rf.Type
				}
			} else {
				fields[fn].(map[string]interface{})[loc] = fv
			}
		}
	}
	e.Fields = fields

	return e, includes
}

func replaceAssetFile(brand string, file interface{}, sysID string, loc string, fmtVideoURL func(string) string) interface{} {
	if originalfileMap, ok := file.(map[string]interface{}); ok {
		// clone map
		fileMap := make(map[string]interface{})
		for k, v := range originalfileMap {
			fileMap[k] = v
		}
		fileName := fileMap["fileName"].(string)
		if fileName != "" {
			url := fileMap["url"].(string)
			if url != "" {
				fn := GetImageFileName(fileName, sysID, loc)
				fileMap["fileName"] = fn
				if IsVideoFile(fileName) {
					fileMap["url"] = fmtVideoURL(fmt.Sprintf("%s/%s", brand, fn))
				} else {
					fileMap["url"] = fmt.Sprintf(imageCDNFmt, cdnClientID, brand, fn)
				}
			}
		}
		return fileMap
	}
	return file
}

func GetAssetImageURL(entry *Entry, imageURLs map[string]string) {
	file, ok := entry.Fields["file"].(map[string]interface{})
	if ok {
		for loc, fc := range file {
			fileContent, ok := fc.(map[string]interface{})
			if ok {
				fileName := fileContent["fileName"].(string)
				if fileName != "" {
					url := fileContent["url"].(string)
					if url != "" {
						imageURLs[GetImageFileName(fileName, entry.Sys.ID, loc)] = fmt.Sprintf("http:%s", url)
					}
				}
			}
		}
	}
}

func downloadImage(URL string) (string, error) {
	resp, err := http.Get(URL)
	if err != nil {
		return "", fmt.Errorf("failed fetch url %s: %s", URL, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to download %s: %d - %s", URL, resp.StatusCode, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body %s: - %s", URL, err.Error())
	}

	return base64.StdEncoding.EncodeToString(bytes.Trim(b, "\xef\xbb\xbf")), nil
}

func GetCloudflareImagesID(repoName string) string {
	cflId := strings.TrimPrefix(repoName, "cms-")
	cflId = strings.TrimPrefix(cflId, "mw-")
	return cflId
}

func IsVideoFile(fileName string) bool {
	ext := path.Ext(fileName)
	vids := []string{".mp4", ".mov", ".webm", ".wmv", ".avi", ".flv", ".avchd"}
	for _, e := range vids {
		if strings.ToLower(ext) == e {
			return true
		}
	}
	return false
}
