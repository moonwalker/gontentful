package gontentful

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"text/template"

	"github.com/lib/pq"
)

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
}

func NewPGSyncSchema(schemaName string, types []*ContentType, items []*Entry) *PGSyncSchema {
	schema := &PGSyncSchema{
		SchemaName: schemaName,
		Tables:     make(map[string]*PGSyncTable, 0),
	}

	for _, item := range items {
		contentType := item.Sys.ContentType.Sys.ID
		switch item.Sys.Type {
		case ENTRY:
			columns := getFieldColumns(types, contentType)
			makeTables(schema.Tables, contentType, columns, item)
			break
		case ASSET:
			columns := []string{"title", "file"}
			makeTables(schema.Tables, "_assets", columns, item)
			break
		case "DeletedEntry":
			// deleted = appendTables(deleted, item, locales)
			break
		case "DeletedAsset":
			// deleted = appendAssets(deleted, assetTableName, item, defaultLocale)
			break
		}
	}

	return schema
}

type rowField struct {
	FieldName  string
	FieldValue interface{}
}

func makeTables(tables map[string]*PGSyncTable, contentType string, columns []string, item *Entry) {
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
				tbl = NewPGSyncTable(tableName, columns)
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

func getFieldColumns(types []*ContentType, contentType string) []string {
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
			fs := getFieldValue(f[i])
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		return pq.Array(arr)

	case []string:
		arr := make([]string, 0)
		for i := 0; i < len(f); i++ {
			fs := getFieldValue(f[i])
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		return pq.Array(arr)

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

func (s *PGSyncSchema) Insert(databaseURL string, initSync bool) error {
	db, _ := sql.Open("postgres", databaseURL)

	_, err := db.Exec(fmt.Sprintf("set search_path='%s'", s.SchemaName))
	if err != nil {
		return err
	}

	if initSync {
		return s.bulkInsert(db)
	}

	return s.deltaInsert(db)
}

func (s *PGSyncSchema) bulkInsert(db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	for _, tbl := range s.Tables {
		if len(tbl.Rows) == 0 {
			continue
		}

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

func (s *PGSyncSchema) deltaInsert(db *sql.DB) error {
	tmpl, err := template.New("").Parse(pgSyncTemplate)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	_, err = db.Exec(buff.String())
	return err
}
