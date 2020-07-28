package gontentful

const pgSyncTemplate = `
{{ range $tblidx, $tbl := .Tables }}
{{ range $itemidx, $item := .Rows }}
INSERT INTO {{ $.SchemaName }}.{{ $tbl.TableName }} (
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
	'{{ .ID }}',
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
{{- end -}}
{{ range $idx, $sys_id := $.Deleted }}
DO $$
DECLARE tn TEXT;
BEGIN
  SELECT table_name INTO tn FROM content._entries WHERE _sys_id = '{{ $sys_id }}';
  IF tn IS NOT NULL THEN
	  EXECUTE 'DELETE FROM content.' || tn || ' WHERE _sys_id = ''{{ $sys_id }}'' CASCADE';
  END IF;
END $$;
{{- end -}}
{{ range $tblidx, $tbl := .ConTables }}
{{ range $rowidx, $row := $tbl.Rows }}
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
{{- end -}}`
