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
	localizedColumns map[string]bool
}

func appendTables(schema *PGSyncSchema, item *Entry, tableName string, fieldColumns []string, refColumns map[string]string, localizedColumns map[string]bool, templateFormat bool) {
	fieldsByLocale := make(map[string][]*rowField, 0)
	defaultLocale := strings.ToLower(schema.DefaultLocale)

	// iterate over fields
	for fieldName, f := range item.Fields {
		locFields, ok := f.(map[string]interface{})
		if !ok {
			continue // no locale, continue
		}

		// snace_case column name
		columnName := toSnakeCase(fieldName)

		// iterate over locale fields
		for _, loc := range schema.Locales {
			// create table
			tbl := schema.Tables[tableName]
			if tbl == nil {
				tbl = newPGSyncTable(tableName, fieldColumns)
				schema.Tables[tableName] = tbl
			}
			locale := strings.ToLower(loc.Code)
			locCode := loc.Code
			if !localizedColumns[columnName] {
				locCode = defaultLocale
			}
			fieldValue := locFields[locCode]
			if sv, ok := fieldValue.(string); fieldValue == nil || (ok && sv == "") {
				continue
			}
			// collect row fields by locale
			fieldsByLocale[locale] = append(fieldsByLocale[locale], &rowField{columnName, fieldValue})
		}
	}

	// append rows with fields to tables
	for locale, rowFields := range fieldsByLocale {
		// table
		tbl := schema.Tables[tableName]
		if tbl != nil {
			appendRowsToTable(item, tbl, rowFields, fieldColumns, templateFormat, schema.ConTables, schema.DeletedConTables, refColumns, tableName, locale)
		}
	}
}

func appendRowsToTable(item *Entry, tbl *PGSyncTable, rowFields []*rowField, fieldColumns []string, templateFormat bool, conTables map[string]*PGSyncConTable, deletedConTables map[string]*PGSyncConTable, refColumns map[string]string, tableName string, locale string) {
	fieldValues := make(map[string]interface{})
	id := fmtSysID(item.Sys.ID, templateFormat, locale)
	fieldValues["_id"] = id
	for _, rowField := range rowFields {
		fieldValues[rowField.fieldName] = convertFieldValue(rowField.fieldValue, templateFormat, locale)
		// append con tables with Array Links
		if _, ok := refColumns[rowField.fieldName]; ok {
			links, ok := rowField.fieldValue.([]interface{})
			if ok {
				addedRefs := make(map[string]bool)
				conTableName := getConTableName(tableName, rowField.fieldName)
				if conTables[conTableName] == nil {
					conTables[conTableName] = &PGSyncConTable{
						TableName: conTableName,
						Columns:   []string{tableName, fmt.Sprintf("%s_sys_id", tableName), refColumns[rowField.fieldName], fmt.Sprintf("%s_sys_id", refColumns[rowField.fieldName]), "_locale"},
						Rows:      make([][]interface{}, 0),
					}
				}
				for _, e := range links {
					f, ok := e.(map[string]interface{})
					if ok {
						sysConID := convertSysID(f, templateFormat)
						conID := convertSys(f, templateFormat, locale)
						if id != "" && conID != "" && !addedRefs[conID] {
							var conRow []interface{}
							if templateFormat {
								conRow = []interface{}{id, fmt.Sprintf("'%s'", item.Sys.ID), conID, sysConID, fmt.Sprintf("'%s'", locale)}
							} else {
								conRow = []interface{}{id, item.Sys.ID, conID, sysConID, locale}
							}
							conTables[conTableName].Rows = append(conTables[conTableName].Rows, conRow)
							addedRefs[conID] = true
						} else {
							fmt.Println(tbl.TableName, id, rowField.fieldName, conID)
						}
					}
				}
			} else {
				conTableName := getConTableName(tableName, rowField.fieldName)
				if deletedConTables[conTableName] == nil {
					deletedConTables[conTableName] = &PGSyncConTable{
						TableName: conTableName,
						Columns:   []string{tableName},
						Rows:      make([][]interface{}, 0),
					}
				}
				if id != "" {
					conRow := []interface{}{id}
					deletedConTables[conTableName].Rows = append(deletedConTables[conTableName].Rows, conRow)
				}
			}
		}
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

	}
	row := newPGSyncRow(item, fieldColumns, fieldValues, locale)
	tbl.Rows = append(tbl.Rows, row)
}

func convertFieldValue(v interface{}, t bool, locale string) interface{} {
	switch f := v.(type) {

	case map[string]interface{}:
		if f["sys"] != nil {
			s := convertSysID(f, t)
			if s != "" {
				return s
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
			if t {
				return fmt.Sprintf("'%s'", string(data))
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

func convertSys(f map[string]interface{}, t bool, locale string) string {
	s, ok := f["sys"].(map[string]interface{})
	if ok {
		if s["type"] == "Link" {
			return fmtSysID(s["id"], t, locale)
		}
	}
	return ""
}

func fmtSysID(id interface{}, t bool, l string) string {
	if t {
		return fmt.Sprintf("'%v_%s'", id, l)
	}
	return fmt.Sprintf("%v_%s", id, l)
}

func convertSysID(f map[string]interface{}, t bool) string {
	s, ok := f["sys"].(map[string]interface{})
	if ok {
		if s["type"] == "Link" {
			if t {
				return fmt.Sprintf("'%v'", s["id"])
			} else {
				return fmt.Sprintf("%v", s["id"])
			}
		}
	}
	return ""
}

func getColumnsByContentType(types []*ContentType) map[string]*columnData {
	typeColumns := make(map[string]*columnData)
	for _, t := range types {
		if typeColumns[t.Sys.ID] == nil {
			fieldColumns, refColumns, locColumns := getContentTypeColumns(t)
			typeColumns[t.Sys.ID] = &columnData{fieldColumns, refColumns, locColumns}
		}
	}
	return typeColumns
}

func getContentTypeColumns(t *ContentType) ([]string, map[string]string, map[string]bool) {
	fieldColumns := make([]string, 0)
	refColumns := make(map[string]string)
	localizedColumns := make(map[string]bool)
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
			if f.Localized && strings.ToLower(f.ID) != "slug" {
				localizedColumns[colName] = true
			}
		}
	}
	return fieldColumns, refColumns, localizedColumns
}
