package gontentful

const pgReferencesTemplate = `
{{ range $idx, $tbl := $.ConTables }}
CREATE TABLE IF NOT EXISTS {{ $tbl.TableName }} (
	_id serial primary key,
	{{- range $colidx, $col := $tbl.Columns }}
	{{- if $colidx -}},{{- end }}
	"{{ .ColumnName }}" TEXT NOT NULL
	{{- end }}
);
{{ end -}}
--
{{ range $idx, $ref := $.References }}
ALTER TABLE IF EXISTS {{ .TableName }} DROP CONSTRAINT IF EXISTS {{ .TableName }}_{{ .ForeignKey }}_fkey;
--
ALTER TABLE IF EXISTS {{ .TableName }}
  ADD CONSTRAINT {{ .TableName }}_{{ .ForeignKey }}_fkey
  FOREIGN KEY ({{ .ForeignKey }})
  REFERENCES {{ .Reference }}
  ON DELETE CASCADE;
--
{{- end -}}
`
