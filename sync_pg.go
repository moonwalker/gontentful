package gontentful

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

const pgSyncTemplate = `
BEGIN;
{{ range $astidx, $ast := .Assets }}
CALL upsert_assets('{{ .Sys.ID }}', '{{ .Fields["title"] }}', '{{ .Fields["description] }}', '{{ .Fields["fileName"] }}', '{{ .Fields["contentType"] }}', '{{ .Fields["url"] }}', {{ .Fields["version"] }},  {{ .Sys.CreatedAt }}, 'sync', {{ .Sys.UpdatedAt }}, 'sync')
CALL pubish_assets('{{ .Sys.ID }}', '{{ .Fields["title"] }}', '{{ .Fields["description] }}', '{{ .Fields["fileName"] }}', '{{ .Fields["contentType"] }}', '{{ .Fields["url"] }}', {{ .Fields["publishedVersion"] }}, {{ .Sys.PublishedAt }}, 'sync')
{{ -end }}
{{ range $tblidx, $tbl := .Tables }}
{{ range $itemidx, $item := .Rows }}
CALL upsert_{{ .TableName }}_{{ $locale }}('{{ .SysID }}',{{ range $k, $v := .Fields }} {{ $v }},{{ end }}, {{ .Version }}, {{ .CreatedAt }}, 'sync', {{ .UpdatedAt }}, 'sync')
CALL pubish_{{ .TableName }}_{{ $locale }}('{{ .SysID }}',{{ range $k, $v := .Fields }} {{ $v }},{{ end }}, {{ .PublishedVersion }}, {{ .PublishedAt }}, 'sync')
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
	TableName string
	Rows      []PGSyncRow
}

type PGSyncSchema struct {
	SchemaName string
	Tables     []PGSyncTable
	Deleted    []PGSyncTable
}

func NewPGSyncSchema(schemaName string, assetTableName string, items []*Entry) PGSyncSchema {
	schema := PGSyncSchema{
		SchemaName: schemaName,
		Tables:     make([]PGSyncTable, 0),
		Deleted:    make([]PGSyncTable, 0),
	}

	tables := make(map[string][]PGSyncRow)
	deleted := make(map[string][]PGSyncRow)

	for _, item := range items {
		tableName := ""
		deletedName := ""
		switch item.Sys.Type {
		case "Entry":
			tableName = item.Sys.ContentType.Sys.ID
			break
		case "Asset":
			tableName = assetTableName
			break
		case "DeletedEntry":
			deletedName = item.Sys.ContentType.Sys.ID
			break
		case "DeletedAsset":
			deletedName = assetTableName
			break
		}

		if tableName != "" {
			rowToUpsert := NewPGSyncRow(item)
			if tables[tableName] == nil {
				tables[tableName] = make([]PGSyncRow, 0)
			}
			tables[tableName] = append(tables[tableName], rowToUpsert)
		}
		if deletedName != "" {
			rowToDelete := NewPGSyncRow(item)
			if deleted[deletedName] == nil {
				deleted[deletedName] = make([]PGSyncRow, 0)
			}
			deleted[tableName] = append(deleted[tableName], rowToDelete)
		}
	}
	for k, r := range tables {
		table := NewPGSyncTable(k, r)
		schema.Tables = append(schema.Tables, table)
	}
	for k, r := range deleted {
		table := NewPGSyncTable(k, r)
		schema.Deleted = append(schema.Deleted, table)
	}

	return schema
}

func NewPGSyncRow(item *Entry) PGSyncRow {
	row := PGSyncRow{
		SysID:            item.Sys.ID,
		Version:          item.Sys.Version,
		PublishedVersion: item.Sys.PublishedVersion,
		CreatedAt:        item.Sys.CreatedAt,
		UpdatedAt:        item.Sys.UpdatedAt,
		PublishedAt:      item.Sys.PublishedAt,
	}
	if item.Fields != nil {
		row.Fields = make(map[string]interface{})
		for k, f := range item.Fields {
			ft := reflect.TypeOf(f)
			if ft != nil {
				fieldType := ft.String()
				if fieldType == "string" {
					row.Fields[k] = fmt.Sprintf("'%s'", strings.ReplaceAll(f.(string), "'", "''"))
					continue
				}
			}
			row.Fields[k] = f
		}
	}
	return row
}

func NewPGSyncTable(tableName string, rows []PGSyncRow) PGSyncTable {
	table := PGSyncTable{
		TableName: tableName,
		Rows:      rows,
	}

	return table
}

func (s *PGSyncSchema) Render() (string, error) {
	tmpl, err := template.New("").Parse(pgSyncTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}
