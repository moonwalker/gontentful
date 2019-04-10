package gontentful

import (
	"bytes"
	"database/sql"
	"net/url"
	"text/template"
)

const MetaQueryFormat = `
SELECT
	name,
	type,
	link_type,
	items,
	is_localized,
	is_required,
	is_disabled,
	is_omitted,
	validations
FROM %s.%s_meta`

const queryTemplate = `
SELECT
{{ range $fieldidx, $field := $.Fields }}
{{ if $fieldidx }}, {{ end }}
{{ if ne $.Locale $.DefaultLocale  }}COALESCE({{ $.TableName }}_{{ $.Locale }}.{{ $field }},{{ $.TableName }}_{{ $.DefaultLocale }}.{{ $field }}) as {{ $field }}
{{ else }}
{{ $field }}
{{ end }}
{{ end -}}
FROM {{ $.SchemaName }}.{{ $.TableName }}_{{ $.DefaultLocale }} {{ $.TableName }}_{{ $.DefaultLocale }}
{{ if ne $.Locale $.DefaultLocale  }} LEFT JOIN {{ $.SchemaName }}.{{ $.TableName }}_{{ $.Locale }} {{ $.TableName }}_{{ $.Locale }} ON {{ $.TableName }}_{{ $.DefaultLocale }}.sysid = {{ $.TableName }}_{{ $.Locale }}.sysid{{ end -}}
{{ if $.Filters }}WHERE
	{{ range $fkey, $fvalue := $.Filters }}

	{{ end -}}
{{ end -}}
{{ if $.Order }}ORDER BY {{ $.Order }}{{ end -}}
`

type PGQuery struct {
	SchemaName    string
	TableName     string
	Locale        string
	DefaultLocale string
	Fields        []string
	Filters       *url.Values
	Order         string
}

func NewPGQuery(schemaName string, tableName string, locale string, defaultLocale string, fields []string, filters url.Values, order string) PGQuery {
	query := PGQuery{
		SchemaName:    schemaName,
		TableName:     tableName,
		Locale:        locale,
		DefaultLocale: defaultLocale,
		Fields:        fields,
		Filters:       &filters,
		Order:         order,
	}
	return query
}

func (s *PGQuery) Exec(databaseURL string) error {
	db, _ := sql.Open("postgres", databaseURL)

	tmpl, err := template.New("").Parse(queryTemplate)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	return txn.Commit()
}
