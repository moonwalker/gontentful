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
{{ range $tblidx, $tbl := .Tables }}
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	sysId,
	{{- range $k := .Fields }}
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
	{{- range $k := .Fields }}
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
	TableName string
	Rows      []PGSyncRow
}

type PGSyncSchema struct {
	SchemaName string
	Tables     []PGSyncTable
	Deleted    []PGSyncTable
}

func NewPGSyncSchema(schemaName string, assetTableName string, space *Space, items []*Entry) PGSyncSchema {
	schema := PGSyncSchema{
		SchemaName: schemaName,
		Tables:     make([]PGSyncTable, 0),
		Deleted:    make([]PGSyncTable, 0),
	}
	tables := make(map[string][]PGSyncRow)
	deleted := make(map[string][]PGSyncRow)

	locales := make([]string, 0)
	defaultLocale := ""
	for _, l := range space.Locales {
		if l.Default {
			defaultLocale = l.Code
		}
		locales = append(locales, l.Code)
	}

	for _, item := range items {
		switch item.Sys.Type {
		case "Entry":
			tables = appendTables(tables, item, locales)
			break
		case "Asset":
			tables = appendAssets(tables, assetTableName, item, defaultLocale)
			break
		case "DeletedEntry":
			deleted = appendTables(deleted, item, locales)
			break
		case "DeletedAsset":
			deleted = appendAssets(deleted, assetTableName, item, defaultLocale)
			break
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

func NewPGSyncRow(item *Entry, locale string) PGSyncRow {
	row := PGSyncRow{
		SysID:            item.Sys.ID,
		Version:          item.Sys.Version,
		PublishedVersion: item.Sys.PublishedVersion,
		CreatedAt:        item.Sys.CreatedAt,
		UpdatedAt:        item.Sys.UpdatedAt,
		PublishedAt:      item.Sys.PublishedAt,
	}
	if row.Version == 0 {
		row.Version = item.Sys.Revision
	}
	if row.PublishedVersion == 0 {
		row.PublishedVersion = row.Version
	}
	if item.Fields != nil {
		row.Fields = make(map[string]interface{})
		for k, f := range item.Fields {
			lf, ok := f.(map[string]interface{})
			if ok {
				f := lf[locale]
				row.Fields[k] = f
				if f != nil {
					ft := reflect.TypeOf(f)
					if ft != nil {
						fieldType := ft.String()
						if fieldType == "string" {
							row.Fields[k] = fmt.Sprintf("'%s'", strings.ReplaceAll(f.(string), "'", "''"))
							continue
						}
					}
				}
			}
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

func appendTables(tables map[string][]PGSyncRow, item *Entry, locales []string) map[string][]PGSyncRow {
	contentType := item.Sys.ContentType.Sys.ID
	for _, loc := range locales {
		locale := fmtLocale(loc)
		tableName := fmt.Sprintf("%s_%s", contentType, locale)

		if tables[tableName] == nil {
			tables[tableName] = make([]PGSyncRow, 0)
		}
		rowToUpsert := NewPGSyncRow(item, loc)
		tables[tableName] = append(tables[tableName], rowToUpsert)
	}
	return tables
}

func appendAssets(tables map[string][]PGSyncRow, assetTableName string, item *Entry, defaultLocale string) map[string][]PGSyncRow {
	if tables[assetTableName] == nil {
		tables[assetTableName] = make([]PGSyncRow, 0)
	}
	rowToUpsert := NewPGSyncRow(item, defaultLocale)
	tables[assetTableName] = append(tables[assetTableName], rowToUpsert)
	return tables
}
