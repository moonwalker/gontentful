package gontentful

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/lib/pq"
	"github.com/mitchellh/mapstructure"
)

func makeTables(item *Entry, baseName string, fieldColumns []string) []*PGSyncTable {
	type rowField struct {
		fieldName  string
		fieldValue interface{}
	}

	tablesByLocale := make(map[string]*PGSyncTable, 0)
	fieldsByLocale := make(map[string][]*rowField, 0)

	// iterate over fields
	for fieldName, f := range item.Fields {
		locFields, ok := f.(map[string]interface{})
		if !ok {
			continue // no locale, continue
		}

		// iterate over locale fields
		for locale, fieldValue := range locFields {
			tableName := fmtTableName(baseName, locale)
			tbl := tablesByLocale[tableName]
			if tbl == nil {
				tbl = newPGSyncTable(tableName, fieldColumns)
				tablesByLocale[tableName] = tbl
			}

			// collect row fields by locale
			fieldsByLocale[locale] = append(fieldsByLocale[locale], &rowField{fieldName, fieldValue})
		}
	}

	// append rows with fields to tables
	for locale, rowFields := range fieldsByLocale {
		tableName := fmtTableName(baseName, locale)
		tbl := tablesByLocale[tableName]
		if tbl != nil {
			fieldValues := make(map[string]interface{}, len(fieldColumns))
			for _, rowField := range rowFields {
				fieldValues[rowField.fieldName] = convertFieldValue(rowField.fieldValue)
				assetFile, ok := fieldValues[rowField.fieldName].(*AssetFile)
				if ok {
					fieldValues["url"] = assetFile.URL
					fieldValues["filename"] = assetFile.FileName
					fieldValues["contenttype"] = assetFile.ContentType
				}
			}
			row := newPGSyncRow(item, fieldColumns, fieldValues)
			tbl.Rows = append(tbl.Rows, row)
		}
	}

	// return tables as simple array, no need to keep them grouped any longer
	tables := make([]*PGSyncTable, 0)
	for tableName, table := range tablesByLocale {
		table.TableName = tableName
		tables = append(tables, table)
	}
	return tables
}

func convertFieldValue(v interface{}) interface{} {
	switch f := v.(type) {

	case map[string]interface{}:
		if f["sys"] != nil {
			s, ok := f["sys"].(map[string]interface{})
			if ok {
				if s["type"] == "Link" {
					return fmt.Sprintf("%v", s["id"])
				}
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
			fs := convertFieldValue(f[i])
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		return pq.Array(arr)

	case []string:
		arr := make([]string, 0)
		for i := 0; i < len(f); i++ {
			fs := convertFieldValue(f[i])
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		return pq.Array(arr)

	}

	return v
}
