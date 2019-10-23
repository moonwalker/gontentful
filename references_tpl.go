package gontentful

const pgReferencesTemplate = `
{{ range $idx, $tbl := $.Schema.ConTables }}
CREATE TABLE IF NOT EXISTS {{ .TableName }} (
	primary key ({{- range $colidx, $col := .Columns }}{{- if $colidx -}},{{- end -}}{{ .ColumnName }}{{- end -}}),
	{{- range $colidx, $col := .Columns }}
	{{- if $colidx -}},{{- end }}
	"{{ .ColumnName }}" TEXT NOT NULL
	{{- end }}
);
{{ end -}}
--
{{ range $idx, $ref := $.Schema.References }}
ALTER TABLE IF EXISTS {{ .TableName }} DROP CONSTRAINT IF EXISTS {{ .ForeignKey }}_fkey;
--
ALTER TABLE IF EXISTS {{ .TableName }}
  ADD CONSTRAINT {{ .ForeignKey }}_fkey
  FOREIGN KEY ({{ .ForeignKey }},_locale)
  REFERENCES {{ .Reference }} (_sys_id,_locale)
  ON DELETE CASCADE;
--
{{- end -}}
`
