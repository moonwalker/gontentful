package gontentful

const pgPublishTemplate = `
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $.TableName }} (
	_id,
	_sys_id,
	{{- range $k, $v := .FieldColumns }}
	{{ $v }},
	{{- end }}
	_locale,
	_version,
	_created_at,
	_created_by,
	_updated_at,
	_updated_by
) VALUES (
	'{{ .SysID }}_{{ .Locale }}',
	'{{ .SysID }}',
	{{- range $k, $v := .FieldColumns }}
	{{ $item.GetFieldValue $v }},
	{{- end }}
	'{{ .Locale }}',
	'{{ .Version }}',
	to_timestamp('{{ .CreatedAt }}','YYYY-MM-DDThh24:mi:ss.mssZ'),
	'sync',
	to_timestamp('{{ .UpdatedAt }}','YYYY-MM-DDThh24:mi:ss.mssZ'),
	'sync'
)
ON CONFLICT (_id) DO UPDATE
SET
	{{- range $k, $v := .FieldColumns }}
	{{ $v }} = EXCLUDED.{{ $v }},
	{{- end }}
	_locale = EXCLUDED._locale,
	_version= EXCLUDED._version,
	_created_at=EXCLUDED._created_at,
	_created_by=EXCLUDED._created_by,
	_updated_at=EXCLUDED._updated_at,
	_updated_by=EXCLUDED._updated_by
;
{{- end -}}
{{ range $conidx, $con := .ConTables }}
{{ range $rowidx, $row := $con.Rows }}
{{ if not $rowidx -}}
DELETE FROM {{ $.SchemaName }}.{{ $con.TableName }} WHERE {{ index $con.Columns 0 }} = {{ (index $row 0) }};
{{ end -}}
INSERT INTO {{ $.SchemaName }}.{{ $con.TableName }} (
	{{- range $k, $v := $con.Columns }}
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
