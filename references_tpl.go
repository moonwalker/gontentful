package gontentful

const pgReferencesTemplate = `
{{ range $idx, $tbl := $.Schema.ConTables }}
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
--
{{ range $idx, $ref := $.Schema.References }}
-- ALTER TABLE IF EXISTS {{ .TableName }} DROP CONSTRAINT IF EXISTS {{ .ForeignKey }}_fkey;
--
-- ALTER TABLE IF EXISTS {{ .TableName }}
--  ADD CONSTRAINT {{ .ForeignKey }}_fkey
--  FOREIGN KEY ({{ .ForeignKey }})
--  REFERENCES {{ .Reference }} (_id)
--  ON DELETE CASCADE;
--
CREATE INDEX IF NOT EXISTS idx_{{ .TableName }}_{{ .ForeignKey }} ON {{ .TableName }}({{ .ForeignKey }});
--
{{- end -}}
`
