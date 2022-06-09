package gontentful

const pgTemplate = `
{{- if $.SchemaName -}}
CREATE SCHEMA IF NOT EXISTS {{ $.SchemaName }};
--
CREATE EXTENSION IF NOT EXISTS unaccent WITH SCHEMA public;
--
{{- end }}
CREATE TABLE IF NOT EXISTS _asset (
	_id text primary key,
	_sys_id text not null,
	title text,
	description text,
	file_name text,
	content_type text,
	url text,
	_locale text not null,
	_version integer not null default 0,
	_created_at timestamp without time zone default now(),
	_created_by text not null,
	_updated_at timestamp without time zone default now(),
	_updated_by text not null
);
CREATE UNIQUE INDEX IF NOT EXISTS _asset__sys_id__locale ON _asset (_sys_id, _locale);
--
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
	"{{ .ColumnName }}" {{ .ColumnType }},
	{{- end }}
	_locale text not null,
	_version integer not null default 0,
	_created_at timestamp without time zone not null default now(),
	_created_by text not null,
	_updated_at timestamp without time zone not null default now(),
	_updated_by text not null
);
--
CREATE UNIQUE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}__sys_id_locale ON {{ $tbl.TableName }}(_sys_id,_locale);
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}__sys_id ON {{ $tbl.TableName }}(_sys_id);
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}__locale ON {{ $tbl.TableName }}(_locale);
{{- range $tbl.Columns -}}
{{- if .IsIndex }}
CREATE INDEX IF NOT EXISTS idx_{{ $tbl.TableName }}_{{ .ColumnName }} ON {{ $tbl.TableName }}({{ .ColumnName }},_locale);
{{ end -}}
{{- end }}
--
{{- end -}}
`
