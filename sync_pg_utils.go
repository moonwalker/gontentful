package gontentful

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
	"github.com/mitchellh/mapstructure"
)

type rowField struct {
	fieldName  string
	fieldValue interface{}
}

type columnData struct {
	fieldColumns     []string
	columnReferences map[string]string
}

func appendTables(schema *PGSyncSchema, item *Entry, tableName string, fieldColumns []string, refColumns map[string]string, templateFormat bool) {
	fieldsByLocale := make(map[string][]*rowField, 0)
	tablesByName := schema.Tables
	conTablesByName := schema.ConTables
	locales := schema.Locales

	// iterate over fields
	for fieldName, f := range item.Fields {
		locFields, ok := f.(map[string]interface{})
		if !ok {
			continue // no locale, continue
		}

		// snace_case column name
		columnName := toSnakeCase(fieldName)

		// iterate over locale fields
		for _, loc := range locales {
			// create table
			tbl := tablesByName[tableName]
			if tbl == nil {
				tbl = newPGSyncTable(tableName, fieldColumns)
				tablesByName[tableName] = tbl
			}
			locale := strings.ToLower(loc.Code)
			fallback := strings.ToLower(loc.FallbackCode)
			fieldValue := locFields[locale]
			if fieldValue == nil {
				if locFields[fallback] != nil {
					fieldValue = locFields[fallback]
				} else {
					fieldValue = locFields[schema.DefaultLocale]
				}
			}

			// collect row fields by locale
			fieldsByLocale[locale] = append(fieldsByLocale[locale], &rowField{columnName, fieldValue})
		}
	}

	// append rows with fields to tables
	for locale, rowFields := range fieldsByLocale {
		// table
		tbl := tablesByName[tableName]
		if tbl != nil {
			appendRowsToTable(item, tbl, rowFields, fieldColumns, templateFormat, conTablesByName, refColumns, tableName, locale)
		}
	}
}

func appendRowsToTable(item *Entry, tbl *PGSyncTable, rowFields []*rowField, fieldColumns []string, templateFormat bool, conTables map[string]*PGSyncConTable, refColumns map[string]string, tableName string, locale string) {
	fieldValues := make(map[string]interface{})
	id := fmt.Sprintf("%s_%s", item.Sys.ID, locale)
	fieldValues["_id"] = id
	for _, rowField := range rowFields {
		fieldValues[rowField.fieldName] = convertFieldValue(rowField.fieldValue, templateFormat, locale)
		assetFile, ok := fieldValues[rowField.fieldName].(*AssetFile)
		if ok {
			url := assetFile.URL
			fileName := assetFile.FileName
			contentType := assetFile.ContentType
			if templateFormat {
				url = fmt.Sprintf("'%s'", url)
				fileName = fmt.Sprintf("'%s'", fileName)
				contentType = fmt.Sprintf("'%s'", contentType)
			}
			fieldValues["url"] = url
			fieldValues["file_name"] = fileName
			fieldValues["content_type"] = contentType
		}
		// append con tables with Array Links
		if refColumns[rowField.fieldName] != "" {
			links, ok := rowField.fieldValue.([]interface{})
			addedRefs := make(map[string]bool)
			if ok {
				sysID := item.Sys.ID
				conTableName := getConTableName(tableName, rowField.fieldName)
				if conTables[conTableName] == nil {
					conTables[conTableName] = &PGSyncConTable{
						TableName: conTableName,
						Columns:   []string{tableName, refColumns[rowField.fieldName]},
						Rows:      make([][]interface{}, 0),
					}
				}
				ffor _, e := range links {
					f, ok := e.(map[string]interface{})
					if ok {
						conSys := convertSys(f, templateFormat)
						conID := fmt.Sprintf("%s_%s", conSys, locale)
						if id != "" && conID != "" && !addedRefs[conSys] {
							conRow := []interface{}{id, conID}
							conTables[conTableName].Rows = append(conTables[conTableName].Rows, conRow)
							addedRefs[conSys] = true
						} else {
							fmt.Println(tbl.TableName, sysID, rowField.fieldName, conSys, locale)
						}
					}
				}
			}
		}
	}
	row := newPGSyncRow(item, fieldColumns, fieldValues, locale)
	tbl.Rows = append(tbl.Rows, row)
}

func convertFieldValue(v interface{}, t bool, locale string) interface{} {
	switch f := v.(type) {

	case map[string]interface{}:
		if f["sys"] != nil {
			s := convertSys(f, t)
			if s != "" {
				return fmt.Sprintf("%s_%s", s, locale)
			}
		} else if f["fileName"] != nil {
			var v *AssetFile
			mapstructure.Decode(f, &v)
			return v
		} else {
			data, err := json.Marshal(f)
			if err != nil {
				log.Fatal("failed to marshal content field")
			}
			return string(data)
		}

	case []interface{}:
		arr := make([]string, 0)
		for i := 0; i < len(f); i++ {
			fs := convertFieldValue(f[i], t, locale)
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		if t {
			return fmt.Sprintf("'{%s}'", strings.ReplaceAll(strings.Join(arr, ","), "'", "\""))
		}
		return pq.Array(arr)

	case []string:
		arr := make([]string, 0)
		for i := 0; i < len(f); i++ {
			fs := convertFieldValue(f[i], t, locale)
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		if t {
			return fmt.Sprintf("'{%s}'", strings.ReplaceAll(strings.Join(arr, ","), "'", "\""))
		}
		return pq.Array(arr)
	case string:
		if t {
			return fmt.Sprintf("'%s'", strings.ReplaceAll(v.(string), "'", "''"))
		}
	}

	return v
}

func convertSys(f map[string]interface{}, t bool) string {
	s, ok := f["sys"].(map[string]interface{})
	if ok {
		if s["type"] == "Link" {
			if t {
				return fmt.Sprintf("'%v'", s["id"])
			}
			return fmt.Sprintf("%v", s["id"])
		}
	}
	return ""
}

func getColumnsByContentType(types []*ContentType) map[string]*columnData {
	typeColumns := make(map[string]*columnData)
	for _, t := range types {
		if typeColumns[t.Sys.ID] == nil {
			fieldColumns := make([]string, 0)
			refColumns := make(map[string]string)
			for _, f := range t.Fields {
				if !f.Omitted {
					colName := toSnakeCase(f.ID)
					fieldColumns = append(fieldColumns, colName)
					if f.Items != nil {
						linkType := getFieldLinkType(f.Items.LinkType, f.Items.Validations)
						if linkType != "" {
							refColumns[colName] = linkType
						}
					}
				}
			}
			typeColumns[t.Sys.ID] = &columnData{fieldColumns, refColumns}
		}
	}
	return typeColumns
}
