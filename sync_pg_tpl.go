package gontentful

const pgSyncTemplate = `
{{ range $tblname, $tbl := .Tables }}
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	_id,
	_sys_id,
	{{- range $k, $v := .FieldColumns }}
	{{ $v }},
	{{- end }}
	_locale,
	_status,
	_version,
	_created_at,
	_created_by,
	_updated_at,
	_updated_by,
	_published_at,
	_published_by
) VALUES (
	'{{ .ID }}',
	'{{ .SysID }}',
	{{- range $k, $v := .FieldColumns }}
	{{ $item.GetFieldValue $v }},
	{{- end }}
	'{{ .Locale }}',
	'{{ .Status }}',
	{{ .Version }},
	to_timestamp('{{ .CreatedAt }}','YYYY-MM-DDThh24:mi:ssZ'),
	'{{ if not .CreatedBy }}sync{{ else }}{{ .CreatedBy }}{{ end }}',
	to_timestamp('{{ .UpdatedAt }}','YYYY-MM-DDThh24:mi:ssZ'),
	'{{ if not .UpdatedBy }}sync{{ else }}{{ .UpdatedBy }}{{ end }}',
	{{ if .PublishedAt }}to_timestamp('{{ .PublishedAt }}','YYYY-MM-DDThh24:mi:ssZ'){{ else }}NULL{{ end }},
	{{ if and .PublishedAt .PublishedBy }}'{{ .PublishedBy }}'{{ else }}NULL{{ end }}
)
ON CONFLICT (_id) DO UPDATE
SET
	{{- range $k, $v := .FieldColumns }}
	{{ $v }} = EXCLUDED.{{ $v }},
	{{- end }}
	_locale = EXCLUDED._locale,
	_status = EXCLUDED._status,
	_version = EXCLUDED._version,
	_updated_at = EXCLUDED._updated_at,
	_updated_by = EXCLUDED._updated_by,
	_published_at = EXCLUDED._published_at,
	_published_by = EXCLUDED._published_by
;
{{- end -}}
{{- end -}}
{{ range $tblname, $tbl := $.Deleted }}
{{ range $idx, $sys_id := .SysIDs }}
DELETE FROM {{ $.SchemaName }}.{{ $tbl.TableName }} WHERE _sys_id = '{{ $sys_id }}' CASCADE
{{- end -}}
{{- end -}}
{{ range $tblidx, $tbl := .DeletedConTables }}
{{ range $rowidx, $row := $tbl.Rows }}
DELETE FROM {{ $.SchemaName }}.{{ $tbl.TableName }} WHERE {{ index $tbl.Columns 0 }} = {{ (index $row 0) }};
{{- end -}}
{{- end -}}
{{ range $tblidx, $tbl := .ConTables }}
{{ $prevId := "" }}
{{ range $rowidx, $row := $tbl.Rows }}
{{if ne $prevId (index $row 0) -}}
DELETE FROM {{ $.SchemaName }}.{{ $tbl.TableName }} WHERE {{ index $tbl.Columns 0 }} = {{ (index $row 0) }};
{{ end -}}
{{ $prevId = (index $row 0) -}}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	{{- range $k, $v := $tbl.Columns }}
	{{- if $k -}},{{- end -}}{{ $v }}
	{{- end }}
) VALUES (
	{{- range $k, $v := $row }}
	{{- if $k -}},{{- end -}}{{ $v }}
	{{- end -}}
);
{{- end -}}
{{- end -}}
`
