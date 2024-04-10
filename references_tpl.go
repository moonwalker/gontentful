package gontentful

const pgReferencesTemplate = `
CREATE TABLE IF NOT EXISTS table_references (
	_id SERIAL primary key,
	tablename TEXT NOT NULL,
	reference TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_table_references on table_references (tablename, reference);
TRUNCATE TABLE table_references;
{{ range $idx, $ref := $.Schema.Dependencies }}
INSERT INTO table_references 
	(tablename, reference) 
VALUES 
	('{{ .TableName }}', '{{ .Reference }}') 
ON CONFLICT (tablename, reference) DO NOTHING;
{{- end }}
--
{{ range $idx, $ref := $.Schema.References }}
ALTER TABLE IF EXISTS {{ .TableName }} DROP CONSTRAINT IF EXISTS {{ .ForeignKey }}_fkey;
--
ALTER TABLE IF EXISTS {{ .TableName }}
	ADD CONSTRAINT {{ .ForeignKey }}_fkey
	FOREIGN KEY ({{ .ForeignKey }})
	REFERENCES {{ .Reference }} (_id)
	ON DELETE {{ if .IsManyToMany -}}CASCADE{{- else -}}SET NULL{{- end -}};
--
CREATE INDEX IF NOT EXISTS idx_{{ .TableName }}_{{ .ForeignKey }} ON {{ .TableName }}({{ .ForeignKey }});
--
{{- end -}}
`
