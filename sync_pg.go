package gontentful

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

const pgSyncTemplate = `
BEGIN
{{ range $tblidx, $tbl := .Tables }};
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	sysId,
	{{- range $k, $v := .Fields }}
	{{ $k }},
	{{- end }}
	version,
	created_at,
	created_by,
	updated_at,
	updated_by
) VALUES (
	'{{ .SysID }}',
	{{- range $k, $v := .Fields }}
	{{ if $v }}{{ $v }}{{ else }}NULL{{ end }},
	{{- end }}
	{{ .Version }},
	'{{ .CreatedAt }}',
	'sync',
	'{{ .UpdatedAt }}',
	'sync'
)
ON CONFLICT (sysId) DO UPDATE
SET
	{{- range $k, $v := .Fields }}
	{{ $k }} = EXCLUDED.{{ $k }},
	{{- end }}
	version = EXCLUDED.version,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
;
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}__publish (
	sysId,
	{{- range $k, $v := .Fields }}
	{{ $k }},
	{{- end }}
	version,
	published_at,
	published_by
) VALUES (
	'{{ .SysID }}',
	{{- range $k, $v := .Fields }}
	{{ if $v }}{{ $v }}{{ else }}NULL{{ end }},
	{{- end }}
	{{ .PublishedVersion }},
	{{ if .PublishedAt }}to_timestamp('{{ .PublishedAt }}','YYYY-MM-DDThh24:mi:ss.mssZ'){{ else }}now(){{ end }},
	'sync'
)
ON CONFLICT (sysId) DO UPDATE
SET
	{{- range $k, $v := .Fields }}
	{{ $k }} = EXCLUDED.{{ $k }},
	{{- end }}
	version = EXCLUDED.version,
	published_at = now(),
	published_by = EXCLUDED.published_by
;
{{ end -}}
{{ end -}}
COMMIT;`

type PGSyncRow struct {
	SysID            string
	Fields           map[string]interface{}
	Version          int
	PublishedVersion int
	CreatedAt        string
	UpdatedAt        string
	PublishedAt      string
}

type PGSyncTable struct {
	TableName    string
	Columns      []string
	FieldColumns []string
	Rows         []*PGSyncRow
}

type PGSyncSchema struct {
	SchemaName string
	Tables     map[string]*PGSyncTable
	Deleted    []*PGSyncTable
}

func NewPGSyncSchema(schemaName string, assetTableName string, space *Space, types []ContentType, items []*Entry) *PGSyncSchema {
	schema := &PGSyncSchema{
		SchemaName: schemaName,
		Tables:     make(map[string]*PGSyncTable, 0),
		Deleted:    make([]*PGSyncTable, 0),
	}

	// locales := make([]string, 0)
	// // defaultLocale := ""
	// for _, l := range space.Locales {
	// 	if l.Default {
	// 		// defaultLocale = l.Code
	// 	}
	// 	locales = append(locales, l.Code)
	// }

	for _, item := range items {
		switch item.Sys.Type {
		case "Entry":
			makeTables(schema.Tables, types, item)
			break
			// case "Asset":
			// 	tables = appendAssets(tables, assetTableName, item, defaultLocale)
			// 	break
			// case "DeletedEntry":
			// 	deleted = c(deleted, item, locales)
			// 	break
			// case "DeletedAsset":
			// 	deleted = appendAssets(deleted, assetTableName, item, defaultLocale)
			// 	break
		}
	}
	// for k, r := range tables {
	// 	table := NewPGSyncTable(k, r)
	// 	schema.Tables = append(schema.Tables, table)
	// }
	// for k, r := range deleted {
	// 	table := NewPGSyncTable(k, r)
	// 	schema.Deleted = append(schema.Deleted, table)
	// }

	return schema
}

type rowField struct {
	FieldName  string
	FieldValue interface{}
}

func makeTables(tables map[string]*PGSyncTable, types []ContentType, item *Entry) {
	contentType := item.Sys.ContentType.Sys.ID
	rowFields := make(map[string][]*rowField)

	for fieldName, field := range item.Fields {
		locFields, ok := field.(map[string]interface{})
		if !ok {
			continue
		}

		for locale, fieldValue := range locFields {
			tableName := fmtTableName(contentType, locale)
			tbl := tables[tableName]
			if tbl == nil {
				fieldColumns := getFieldColumns(types, contentType)
				tbl = NewPGSyncTable(tableName, fieldColumns)
				tables[tableName] = tbl
			}

			rowFields[locale] = append(rowFields[locale], &rowField{fieldName, fieldValue})
		}
	}

	for locale, rows := range rowFields {
		tableName := fmtTableName(contentType, locale)
		tbl := tables[tableName]
		if tbl != nil {
			row := NewPGSyncRow(item, tbl.FieldColumns, rows)
			tbl.Rows = append(tbl.Rows, row)
		}
	}
}

func fmtTableName(contentType string, locale string) string {
	return fmt.Sprintf("%s_%s", strings.ToLower(contentType), fmtLocale(locale))
}

func getFieldColumns(types []ContentType, contentType string) []string {
	fieldColumns := make([]string, 0)

	for _, t := range types {
		if t.Sys.ID == contentType {
			for _, f := range t.Fields {
				fieldColumns = append(fieldColumns, strings.ToLower(f.ID))
			}
		}
	}

	return fieldColumns
}

func NewPGSyncTable(tableName string, fieldColumns []string) *PGSyncTable {
	columns := []string{"sysid"}
	columns = append(columns, fieldColumns...)
	columns = append(columns, "version", "created_at", "created_by", "updated_at", "updated_by")

	return &PGSyncTable{
		TableName:    tableName,
		Columns:      columns,
		FieldColumns: fieldColumns,
		Rows:         make([]*PGSyncRow, 0),
	}
}

func NewPGSyncRow(item *Entry, fieldColumns []string, rowFields []*rowField) *PGSyncRow {
	row := &PGSyncRow{
		SysID:            item.Sys.ID,
		Fields:           make(map[string]interface{}, len(fieldColumns)),
		Version:          item.Sys.Version,
		CreatedAt:        item.Sys.CreatedAt,
		UpdatedAt:        item.Sys.UpdatedAt,
		PublishedVersion: item.Sys.PublishedVersion,
		PublishedAt:      item.Sys.PublishedAt,
	}
	if row.Version == 0 {
		row.Version = item.Sys.Revision
	}
	if row.PublishedVersion == 0 {
		row.PublishedVersion = row.Version
	}
	for _, fieldCol := range fieldColumns {
		row.Fields[fieldCol] = nil
	}
	for _, rowField := range rowFields {
		row.Fields[rowField.FieldName] = getFieldValue(rowField.FieldValue)
	}
	return row
}

func getFieldValue(v interface{}) interface{} {
	switch f := v.(type) {
	case map[string]interface{}:
		if f["sys"] != nil {
			s, ok := f["sys"].(map[string]interface{})
			if ok {
				if s["type"] == "Link" {
					return fmt.Sprintf("%v", s["id"])
				}
			}
		}

	case []interface{}:
		arr := make([]string, 0)
		for i := 0; i < len(f); i++ {
			fs := getFieldValue(f[i])
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		return arr
	}

	return v
}

func (r *PGSyncRow) Values(fieldColumns []string) []interface{} {
	values := []interface{}{
		r.SysID,
	}
	for _, fieldName := range fieldColumns {
		values = append(values, r.Fields[fieldName])
	}
	return append(values, r.Version, r.CreatedAt, "sync", r.UpdatedAt, "sync")
}

func (s *PGSyncSchema) BulkInsert(databaseURL string) error {
	db, _ := sql.Open("postgres", databaseURL)

	_, err := db.Exec(fmt.Sprintf("set search_path='%s'", s.SchemaName))
	if err != nil {
		return err
	}

	txn, err := db.Begin()
	if err != nil {
		return err
	}

	for _, tbl := range s.Tables {
		if len(tbl.Rows) == 0 {
			continue
		}

		// log.Printf("table: %s, rows: %d\n", tbl.TableName, len(tbl.Rows))

		stmt, err := txn.Prepare(pq.CopyIn(tbl.TableName, tbl.Columns...))
		if err != nil {
			return err
		}

		for _, row := range tbl.Rows {
			_, err = stmt.Exec(row.Values(tbl.FieldColumns)...)
			if err != nil {
				return err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}

		err = stmt.Close()
		if err != nil {
			return err
		}
	}

	return txn.Commit()
}

// func appendTables(tables map[string][]PGSyncRow, item *Entry, locales []string) map[string][]PGSyncRow {
// 	contentType := item.Sys.ContentType.Sys.ID
// 	for _, loc := range locales {
// 		locale := fmtLocale(loc)
// 		tableName := fmt.Sprintf("%s_%s", strings.ToLower(contentType), locale)

// 		if tables[tableName] == nil {
// 			tables[tableName] = make([]PGSyncRow, 0)
// 		}
// 		rowToUpsert := NewPGSyncRow(item, loc)
// 		tables[tableName] = append(tables[tableName], rowToUpsert)
// 	}
// 	return tables
// }

// func appendAssets(tables map[string][]PGSyncRow, assetTableName string, item *Entry, defaultLocale string) map[string][]PGSyncRow {
// 	if tables[assetTableName] == nil {
// 		tables[assetTableName] = make([]PGSyncRow, 0)
// 	}
// 	rowToUpsert := NewPGSyncRow(item, defaultLocale)
// 	tables[assetTableName] = append(tables[assetTableName], rowToUpsert)
// 	return tables
// }

func evaluateField(field interface{}) interface{} {
	// ft := reflect.TypeOf(field)
	// fmt.Println(">>>", ft)

	a, ok := field.([]interface{})
	if ok {
		for i := 0; i < len(a); i++ {
			// fmt.Println(a[i])
		}
	} else {
		return field
	}

	return nil

	// if field != nil {
	// 	ft := reflect.TypeOf(field)
	// 	if ft != nil {
	// 		fieldType := ft.String()

	// 		fmt.Println(fieldType)
	// 		if fieldType == "integer" {
	// 			return fmt.Sprintf("%v", field)
	// 		} else if fieldType == "string" {
	// 			return fmt.Sprintf("'%s'", strings.ReplaceAll(field.(string), "'", "''"))
	// 		} else if strings.HasPrefix(fieldType, "[]") {
	// 			arr := make([]string, 0)
	// 			a, ok := field.([]interface{})
	// 			if ok {
	// 				for i := 0; i < len(a); i++ {
	// 					fs := evaluateField(a[i])
	// 					if fs != "" {
	// 						arr = append(arr, fs)
	// 					}
	// 				}
	// 			}
	// 			return strings.Join(arr, ",") // fmt.Sprintf("ARRAY[%s]", strings.Join(arr, ","))
	// 		} else if strings.HasPrefix(fieldType, "map[string]") {
	// 			e, ok := field.(map[string]interface{})
	// 			if ok && e["sys"] != nil {
	// 				s, ok := e["sys"].(map[string]interface{})
	// 				if ok {
	// 					if s["type"] == "Link" {
	// 						return fmt.Sprintf("'%v'", s["id"])
	// 					}
	// 				}
	// 			}
	// 			data, err := json.Marshal(e)
	// 			if err != nil {
	// 				fmt.Println(fieldType, " Marshal ERROR.", field, err)
	// 				return "'{}'"
	// 			}
	// 			return fmt.Sprintf("'%s'", string(data))
	// 		}
	// 		return fmt.Sprintf("%v", field)
	// 	}
	// }
	// return ""
}
