package gontentful

const pgTemplate = `
{{- if not $.ContentTypePublish }}
{{- if $.SchemaName -}}
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;
--
{{- end }}
CREATE TABLE IF NOT EXISTS {{ $.AssetTable.Name }} (
	_id text primary key,
	_sys_id text not null,
	{{- range $colidx, $col := $.AssetTable.Columns }}
	"{{ $col }}" text,
	{{- end }}
	_locale text not null,
	_status text not null,
	_version integer not null default 0,
	_created_at timestamp without time zone default now(),
	_created_by text not null,
	_updated_at timestamp without time zone default now(),
	_updated_by text not null,
	_published_at timestamp without time zone,
	_published_by text
);
CREATE UNIQUE INDEX IF NOT EXISTS {{ $.AssetTable.Name }}__sys_id__locale ON {{ $.AssetTable.Name }} (_sys_id, _locale);
--
CREATE TABLE IF NOT EXISTS {{ $.SchemaTableName }} (
	table_name text primary key,
	model text not null unique,
	name text not null unique,
	description text,
	displayField text not null,
	fields jsonb not null default '[]'::jsonb,
	_version integer not null default 0,
	_created_at timestamp without time zone default now(),
	_created_by text not null,
	_updated_at timestamp without time zone default now(),
	_updated_by text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS {{ $.SchemaTableName }}_model ON {{ $.SchemaTableName }} (model);
--
INSERT INTO {{ $.SchemaTableName }} (
	table_name,
	model,
	name,
	description,
	displayField,
	fields,
	_created_by,
	_updated_by
) VALUES (
	'{{ $.AssetTable.Name }}',
	'{{ $.AssetTable.Name }}',
	'{{ $.AssetTable.FieldName }}',
	'{{ $.AssetTable.FieldName }}',
	'{{ $.AssetTable.DisplayField }}',
	'{{ $.AssetTable.Fields | marshal }}'::jsonb,
	'sync',
	'sync'
);
--
{{ end -}}
{{ range $tblidx, $tbl := $.Tables }}
--
{{- if $.DropTables }}
DROP TABLE IF EXISTS {{ $tbl.TableName }} CASCADE;
{{ end -}}
--
CREATE TABLE IF NOT EXISTS {{ $tbl.TableName }} (
	_id text primary key,
	_sys_id text not null,
	{{- range $colidx, $col := $tbl.Columns }}
	"{{ .ColumnName }}" {{ .ColumnType }}{{ if .Required }} not null{{ end }},
	{{- end }}
	_locale text not null,
	_status text not null,
	_version integer not null default 0,
	_created_at timestamp without time zone not null default now(),
	_created_by text not null,
	_updated_at timestamp without time zone not null default now(),
	_updated_by text not null,
	_published_at timestamp without time zone,
	_published_by text
);
--
CREATE UNIQUE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}__sys_id_locale ON {{ $tbl.TableName }}(_sys_id,_locale);
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}__sys_id ON {{ $tbl.TableName }}(_sys_id);
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}__locale ON {{ $tbl.TableName }}(_locale);
{{- range $tbl.Columns -}}
{{- if .IsIndex }}
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}_{{ .ColumnName }} ON {{ $tbl.TableName }}({{ .ColumnName }});
{{- if .IsUnique }}
CREATE UNIQUE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}_{{ .ColumnName }}_locale ON {{ $tbl.TableName }}({{ .ColumnName }},_locale);
{{- else -}}
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}_{{ .ColumnName }}_locale ON {{ $tbl.TableName }}({{ .ColumnName }},_locale);
{{ end -}}
{{ end -}}
{{- end }}
--
INSERT INTO {{ $.SchemaTableName }} (
	table_name,
	model,
	name,
	description,
	displayField,
	fields,
	_version,
	_created_at,
	_created_by,
	_updated_at,
	_updated_by
) VALUES (
	'{{ $tbl.TableName }}',
	'{{ $tbl.Schema.ID }}',
	'{{ $tbl.Schema.Name }}',
	'{{ $tbl.Schema.Description }}',
	'{{ $tbl.Schema.DisplayField }}',
	'{{ $tbl.Schema.Fields | marshal }}'::jsonb,
	{{ $tbl.Schema.Version }},
	to_timestamp('{{ $tbl.Schema.CreatedAt }}','YYYY-MM-DDThh24:mi:ssZ'),
	'{{ if $tbl.Schema.CreatedBy }}{{ $tbl.Schema.CreatedBy }}{{ else }}sync{{ end }}',
	to_timestamp('{{ $tbl.Schema.UpdatedAt }}','YYYY-MM-DDThh24:mi:ssZ'),
	'{{ if $tbl.Schema.UpdatedBy }}{{ $tbl.Schema.UpdatedBy }}{{ else }}sync{{ end }}'
)
ON CONFLICT (table_name) DO UPDATE
SET
	name = EXCLUDED.name,
	description = EXCLUDED.description,
	displayField = EXCLUDED.displayField,
	fields = EXCLUDED.fields,
	_version = EXCLUDED._version,
	_updated_at = EXCLUDED._updated_at,
	_updated_by = EXCLUDED._updated_by
;
--
{{- end -}}
--
{{ range $idx, $tbl := $.ConTables }}
CREATE TABLE IF NOT EXISTS {{ .TableName }} (
	_id SERIAL primary key,
	{{- range $colidx, $col := .Columns }}
	{{- if $colidx -}},{{- end }}
	"{{ .ColumnName }}" TEXT NOT NULL
	{{- end }}
);
{{ range $idxn, $idxf := .Indices }}
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}_{{ $tbl.TableName }}_{{ $idxn }} ON {{ $tbl.TableName }} ({{ $idxf }});
{{- end }}
{{ end -}}
`
