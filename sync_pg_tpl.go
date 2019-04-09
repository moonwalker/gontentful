package gontentful

const pgSyncTemplate = `
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
{{ end -}}`
