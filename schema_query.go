package gontentful

import (
	"bytes"
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
SELECT row_to_json(f) AS fields
FROM (
	SELECT
	{{ range $fieldidx, $field := $.Fields }}
		{{ if .Localized }}
			{{ if $.Locale != $.DefaultLocale && $.Locale != "" }}COALESCE({{ end -}}
			{{ if $.Locale != "" }}fields -> '{{ .Name }}' -> '{{ if $.Locale }}'{{ end -}}
			{{ if $.Locale != $.DefaultLocale }}{{ if $.Locale != "" }}, {{ end -}}fields -> '{{ .Name }}' -> '{{ if $.DefaultLocale }}'{{ if $.Locale != "" }}){{ end -}}{{ end -}}
		{{ else }}
			fields -> '{{ .Name }}'
		{{ end -}}
		as {{ .Name }}
	{{ end -}}
	FROM {{ $.SchemaName }}.{{ $.TableName }}
	WHERE id = {{ $.TableName }}.id
) f)
FROM {{ $.SchemaName }}.{{ $.TableName }}
{{ if $.Filters != nil }}WHERE
	{{ range $fkey, $fvalue := $.Filters }}

	{{ end -}}
{{ end -}}
{{ if $.Order != "" }}ORDER BY {{ $.Order }}{{ end -}}
`

type DBContentQuery struct {
	SchemaName    string
	TableName     string
	Locale        string
	DefaultLocale string
	Fields        []*PGJSONBMetaRow
	Filters       *url.Values
	Order         string
}

func NewDBContentQuery(schemaName string, tableName string, locale string, defaultLocale string, fields []*PGJSONBMetaRow, filters url.Values, order string) DBContentQuery {
	query := DBContentQuery{
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

func (s *DBContentQuery) Render() (string, error) {
	tmpl, err := template.New("").Parse(queryTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}
