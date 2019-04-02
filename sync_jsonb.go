package gontentful

import (
	"bytes"
	"encoding/json"
	"text/template"
)

const jsonbSyncTemplate = `BEGIN;
CREATE SCHEMA IF NOT EXISTS {{ .SchemaName }};
COMMIT;
--
{{ range $tblidx, $tbl := .Tables }}
BEGIN;
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ .TableName }} (
	id text primary key not null,
	fields jsonb not null default '[]'::jsonb,
	type text not null,
	revision integer not null default 0,
	version integer not null default 0,
	published_version integer not null default 0,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null,
	published_at timestamp without time zone,
	published_by text
);
--
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	id,
	fields,
	type,
	revision,
	version,
	published_version,
	created_at,
	created_by,
	updated_at,
	updated_by,
	published_at,
	published_by
) VALUES (
	'{{ .ID }}',
	{{ if .Fields }}'{{ .Fields }}'::jsonb{{ else }}NULL{{ end }},
	'{{ .Type }}',
	{{ .Revision }},
	{{ .Version }},
	{{ .PublishedVersion }},
	to_timestamp('{{ .CreatedAt }}', 'YYYY-MM-DDThh24:mi:ss.mssZ'),
	'system',
	to_timestamp('{{ .UpdatedAt }}', 'YYYY-MM-DDThh24:mi:ss.mssZ'),
	'system',
	{{ if .PublishedAt }}to_timestamp('{{ .PublishedAt }}','YYYY-MM-DDThh24:mi:ss.mssZ'){{ else }}NULL{{ end }},
	'{{ .PublishedBy }}'
)
ON CONFLICT (id) DO UPDATE
SET
	fields = EXCLUDED.fields,
	type = EXCLUDED.type,
	revision = EXCLUDED.revision,
	version = EXCLUDED.version,
	published_version = EXCLUDED.published_version,
	updated_at = EXCLUDED.updated_at,
	updated_by = EXCLUDED.updated_by,
	published_at = EXCLUDED.published_at,
	published_by = EXCLUDED.published_by
RETURNING 1;
--
{{ end -}}
COMMIT;
{{ end -}}`

type SyncJSONBRow struct {
	ID               string `json:"id,omitempty"`
	Fields           string `json:"fields,omitempty"`
	Type             string `json:"type,omitempty"`
	Version          int    `json:"version,omitempty"`
	Revision         int    `json:"revision,omitempty"`
	PublishedVersion int    `json:"publishedVersion,omitempty"`
	CreatedAt        string `json:"createdAt,omitempty"`
	CreatedBy        string `json:"createdBy,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
	UpdatedBy        string `json:"updatedBy,omitempty"`
	PublishedAt      string `json:"publishedAt,omitempty"`
	PublishedBy      string `json:"publishedBy,omitempty"`
}

type SyncJSONBTable struct {
	TableName string
	Rows      []SyncJSONBRow
}

type SyncJSONBSchema struct {
	SchemaName     string
	AssetTableName string
	Tables         []SyncJSONBTable
	Deleted        []SyncJSONBTable
}

func NewSyncJSONBSchema(schemaName string, assetTableName string, items []*Entry) SyncJSONBSchema {
	schema := SyncJSONBSchema{
		SchemaName: schemaName,
		Tables:     make([]SyncJSONBTable, 0),
		Deleted:    make([]SyncJSONBTable, 0),
	}

	tables := make(map[string][]SyncJSONBRow)
	deleted := make(map[string][]SyncJSONBRow)

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
			rowToUpsert := NewSyncJSONBRow(item)
			if tables[tableName] == nil {
				tables[tableName] = make([]SyncJSONBRow, 0)
			}
			tables[tableName] = append(tables[tableName], rowToUpsert)
		}
		if deletedName != "" {
			rowToDelete := NewSyncJSONBRow(item)
			if deleted[deletedName] == nil {
				deleted[deletedName] = make([]SyncJSONBRow, 0)
			}
			deleted[tableName] = append(deleted[tableName], rowToDelete)
		}
	}
	for k, r := range tables {
		table := NewSyncJSONBTable(k, r)
		schema.Tables = append(schema.Tables, table)
	}
	for k, r := range deleted {
		table := NewSyncJSONBTable(k, r)
		schema.Deleted = append(schema.Deleted, table)
	}

	return schema
}

func NewSyncJSONBRow(item *Entry) SyncJSONBRow {
	row := SyncJSONBRow{
		ID:               item.Sys.ID,
		Type:             item.Sys.Type,
		Version:          item.Sys.Version,
		Revision:         item.Sys.Revision,
		PublishedVersion: item.Sys.PublishedVersion,
		CreatedAt:        item.Sys.CreatedAt,
		CreatedBy:        "system",
		UpdatedAt:        item.Sys.UpdatedAt,
		UpdatedBy:        "system",
		PublishedAt:      item.Sys.PublishedAt,
		PublishedBy:      "",
	}
	if item.Fields != nil {
		f, err := json.Marshal(item.Fields)
		if err == nil {
			row.Fields = formatText(string(f))
		}
	}
	if item.Sys.CreatedBy != nil {
		row.CreatedBy = (*item.Sys.CreatedBy).Sys.ID
	}
	if item.Sys.UpdatedBy != nil {
		row.PublishedBy = (*item.Sys.UpdatedBy).Sys.ID
	}
	if item.Sys.PublishedBy != nil {
		row.PublishedBy = (*item.Sys.PublishedBy).Sys.ID
	}

	return row
}

func NewSyncJSONBTable(tableName string, rows []SyncJSONBRow) SyncJSONBTable {
	table := SyncJSONBTable{
		TableName: tableName,
		Rows:      rows,
	}

	return table
}

func (s *SyncJSONBSchema) Render() (string, error) {
	tmpl, err := template.New("").Parse(jsonbSyncTemplate)
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
