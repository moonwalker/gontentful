package gontentful

const pgSyncTemplate = `
{{ range $tblidx, $tbl := .Tables }}
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	_sys_id
	{{- range $k, $v := .FieldColumns }}
	,{{ $v }}
	{{- end }}
	{{- range $k, $v := .MetaColumns }}
	,{{ $v }}
	{{- end }}
) VALUES (
	'{{ .SysID }}'
	{{- range $k, $v := .FieldColumns }}
	,{{ $item.GetFieldValue $v }}
	{{- end }}
	{{- range $k, $v := .MetaColumns }}
	,{{ $item.GetFieldValue $v }}
	{{- end }}
)
ON CONFLICT (sys_id) DO UPDATE
SET
	{{- range $k, $v := .FieldColumns }}
	{{ if $k }},{{- end }}{{ $v }} = EXCLUDED.{{ $v }}
	{{- end }}
	{{- range $k, $v := .MetaColumns }}
	,{{ $v }} = EXCLUDED.{{ $v }}
	{{- end }}
;
{{- end -}}
{{- end -}}
{{ range $idx, $sys_id := $.Deleted }}
DO $$
DECLARE tn TEXT;
BEGIN
  SELECT table_name INTO tn FROM content._entries WHERE _sys_id = '{{ $sys_id }}';
  IF tn IS NOT NULL THEN
	  EXECUTE 'DELETE FROM content.' || tn || ' WHERE _sys_id = ''{{ $sys_id }}''';
  END IF;
END $$;
{{- end -}}
{{ range $tblidx, $tbl := .ConTables }}
{{ range $rowidx, $row := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
	{{- range $k, $v := .Columns }}
	{{- if $k -}},{{- end -}}{{ $v }}
	{{- end }}
) VALUES (
	{{- range $k, $v := $row }}
	{{- if $k -}},{{- end -}}{{ $v }}
)
;
{{- end -}}
{{- end -}}`
