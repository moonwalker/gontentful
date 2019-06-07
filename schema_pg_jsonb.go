// $ ... | docker exec -i <containerid> psql -U postgres

package gontentful

import (
	"bytes"
	"encoding/json"
	"strings"
	"text/template"
)

const jsonbTemplate = `BEGIN;
CREATE SCHEMA IF NOT EXISTS {{ .SchemaName }};
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $.AssetTableName }} (
	id text primary key not null,
	fields jsonb,
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ $.ModelsTableName }} (
	id text primary key not null,
	name text not null,
	description text,
	display_field text not null,
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
{{ range $tblidx, $tbl := .Tables }}
INSERT INTO {{ $.SchemaName }}.{{ $.ModelsTableName }} (
	id,
	name,
	description,
	display_field,
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
	'{{ .TableName }}',
	'{{ .Name }}',
	'{{ .Description }}',
	'{{ .DisplayField }}',
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
	name = EXCLUDED.name,
	description = EXCLUDED.description,
	display_field = EXCLUDED.display_field,
	revision = EXCLUDED.revision,
	version = EXCLUDED.version,
	published_version = EXCLUDED.published_version,
	updated_at = EXCLUDED.updated_at,
	updated_by = EXCLUDED.updated_by,
	published_at = EXCLUDED.published_at,
	published_by = EXCLUDED.published_by
RETURNING 1;
--
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
CREATE TABLE IF NOT EXISTS {{ $.SchemaName }}.{{ .TableName }}_meta (
	name text primary key not null,
	type text not null,
	link_type text,
	items jsonb,
	is_localized boolean default false,
	is_required boolean default false,
	is_unique boolean default false,
	is_disabled boolean default false,
	is_omitted boolean default false,
	validations jsonb,
	created_at timestamp without time zone default now(),
	created_by text not null,
	updated_at timestamp without time zone default now(),
	updated_by text not null
);
--
{{ range $fieldsidx, $fields := .Metas }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }}_meta (
	name,
	type,
	link_type,
	items,
	is_localized,
	is_required,
	is_disabled,
	is_omitted,
	validations,
	created_by,
	updated_by
) VALUES (
	'{{ .Name }}',
	'{{ .Type }}',
	'{{ .LinkType }}',
	{{ if .Items }}'{{ .Items }}'::jsonb{{ else }}NULL{{ end }},
	{{ .Localized }},
	{{ .Required }},
	{{ .Disabled }},
	{{ .Omitted }},
	{{ if .Validations }}'{{ .Validations }}'::jsonb{{ else }}NULL{{ end }},
	'system',
	'system'
)
ON CONFLICT (name) DO UPDATE
SET
	type = EXCLUDED.type,
	link_type = EXCLUDED.link_type,
	items = EXCLUDED.items,
	is_localized = EXCLUDED.is_localized,
	is_required = EXCLUDED.is_required,
	is_disabled = EXCLUDED.is_disabled,
	is_omitted = EXCLUDED.is_omitted,
	validations = EXCLUDED.validations,
	updated_at = now(),
	updated_by = EXCLUDED.updated_by
RETURNING 1;
--
{{ end -}}{{ end -}}
COMMIT;`

type PGJSONBModelTable struct {
	TableName        string            `json:"tableName,omitempty"`
	Name             string            `json:"name,omitempty"`
	Description      string            `json:"description,omitempty"`
	DisplayField     string            `json:"displayField,omitempty"`
	Version          int               `json:"version,omitempty"`
	Revision         int               `json:"revision,omitempty"`
	PublishedVersion int               `json:"publishedVersion,omitempty"`
	CreatedAt        string            `json:"createdAt,omitempty"`
	CreatedBy        string            `json:"createdBy,omitempty"`
	UpdatedAt        string            `json:"updatedAt,omitempty"`
	UpdatedBy        string            `json:"updatedBy,omitempty"`
	PublishedAt      string            `json:"publishedAt,omitempty"`
	PublishedBy      string            `json:"publishedBy,omitempty"`
	Metas            []*PGJSONBMetaRow `json:"metas,omitempty"`
}

type PGJSONBMetaRow struct {
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	LinkType  string `json:"linkType,omitempty"`
	Items     string `json:"version,omitempty"`
	Required  bool   `json:"required,omitempty"`
	Localized bool   `json:"localized,omitempty"`
	// Unique      bool   `json:"unique,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
	Omitted     bool   `json:"omitted,omitempty"`
	Validations string `json:"validations,omitempty"`
}

type PGJSONBSchema struct {
	SchemaName      string
	AssetTableName  string
	ModelsTableName string
	Tables          []*PGJSONBModelTable
}

func NewPGJSONBSchema(schemaName string, items []*ContentType) *PGJSONBSchema {
	schema := &PGJSONBSchema{
		SchemaName:      schemaName,
		AssetTableName:  assetTableName,
		ModelsTableName: "_models",
		Tables:          make([]*PGJSONBModelTable, 0),
	}

	for _, item := range items {
		table := NewPGJSONBModelTable(item)
		schema.Tables = append(schema.Tables, table)
	}

	return schema
}

func NewPGJSONBModelTable(item *ContentType) *PGJSONBModelTable {
	table := &PGJSONBModelTable{
		TableName:    item.Sys.ID,
		Name:         formatText(item.Name),
		Description:  formatText(item.Description),
		DisplayField: item.DisplayField,
		Revision:     item.Sys.Revision,
		CreatedAt:    item.Sys.CreatedAt,
		UpdatedAt:    item.Sys.UpdatedAt,
		Metas:        make([]*PGJSONBMetaRow, 0),
	}

	for _, field := range item.Fields {
		meta := NewPGJSONBMetaRow(field)
		table.Metas = append(table.Metas, meta)
	}

	return table
}

func NewPGJSONBMetaRow(field *ContentTypeField) *PGJSONBMetaRow {
	meta := &PGJSONBMetaRow{
		Name:      formatText(field.Name),
		Type:      field.Type,
		LinkType:  field.LinkType,
		Required:  field.Required,
		Localized: field.Localized,
		Disabled:  field.Disabled,
		Omitted:   field.Omitted,
	}
	if field.Items != nil {
		i, err := json.Marshal(field.Items)
		if err == nil {
			meta.Items = formatText(string(i))
		}
	}

	if field.Validations != nil {
		v, err := json.Marshal(field.Validations)
		if err == nil {
			meta.Validations = formatText(string(v))
		}
	}

	return meta
}

func (s *PGJSONBSchema) Render() (string, error) {
	tmpl, err := template.New("").Parse(jsonbTemplate)
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

func formatText(text string) string {
	return strings.ReplaceAll(text, "'", "''")
}
