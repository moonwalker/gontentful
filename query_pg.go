package gontentful

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"text/template"
)

const MetaQueryFormat = `
SELECT
	name,
--	type,
--	link_type,
--	items,
	is_localized
--	is_required,
--  is_disabled,
--	is_omitted,
--	validations
FROM %s.%s__meta`

const queryTemplate = `
SELECT
{{- range $fieldidx, $field := $.Columns }}
{{- if $fieldidx }},{{- end }}
{{ if and .Localized (ne $.Locale $.DefaultLocale)  }}COALESCE({{ $.TableName }}_{{ $.Locale }}.{{ .Name }},{{ $.TableName }}_{{ $.DefaultLocale }}.{{ .Name }}) as {{ .Name }}
{{- else }}{{ $.TableName }}_{{ $.DefaultLocale }}.{{ .Name }} as {{ .Name }}
{{- end -}}
{{- end }}
FROM {{ $.SchemaName }}.{{ $.TableName }}_{{ $.DefaultLocale }} {{ $.TableName }}_{{ $.DefaultLocale }}
{{ if ne $.Locale $.DefaultLocale }}LEFT JOIN {{ $.SchemaName }}.{{ $.TableName }}_{{ $.Locale }} {{ $.TableName }}_{{ $.Locale }} ON {{ $.TableName }}_{{ $.DefaultLocale }}.sysid = {{ $.TableName }}_{{ $.Locale }}.sysid{{ end }}
{{ if $.Filters }}WHERE
{{- range $fkey, $fvalue := $.Filters }}

{{- end -}}
{{- end -}}
{{ if $.Order }}ORDER BY {{ $.Order }}{{ end -}}
`

var (
	comparerRegex = regexp.MustCompile(`.+\[(^\])+]=.+`)
)

type PQQueryFilter struct {
	Field    string
	Comparer string
	Value    string
}

type PGQuery struct {
	SchemaName    string
	TableName     string
	Locale        string
	DefaultLocale string
	Fields        []string
	Filters       []*PQQueryFilter
	Order         string
	Columns       []*PGSQLMeta
	Limit         int
	Skip          int
	Include       int
}

func NewPGQuery(schemaName string, tableName string, locale string, defaultLocale string, fields []string, filters url.Values, order string, skip int, limit int, include int) PGQuery {
	query := PGQuery{
		SchemaName:    schemaName,
		TableName:     tableName,
		Locale:        locale,
		DefaultLocale: defaultLocale,
		Fields:        fields,
		Filters:       formatFilter(filters),
		Order:         formatOrder(order),
		Skip:          skip,
		Limit:         limit,
		Include:       include,
	}
	return query
}

func (s *PGQuery) Exec(databaseURL string) error {
	db, _ := sql.Open("postgres", databaseURL)

	metas, err := db.Query(fmt.Sprintf(MetaQueryFormat, s.SchemaName, s.TableName))
	if err != nil {
		log.Fatal(err)
	}
	defer metas.Close()
	fields := make(map[string]struct{})
	for _, f := range s.Fields {
		fields[f] = struct{}{}
	}
	allFields := (s.Fields == nil)
	s.Columns = make([]*PGSQLMeta, 0)
	for metas.Next() {
		meta := PGSQLMeta{}
		err := metas.Scan(&meta.Name, &meta.Localized)
		if err != nil {
			return err
		}
		_, ok := fields[meta.Name]
		if allFields || ok {
			s.Columns = append(s.Columns, &meta)
		}
	}

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

	fmt.Println(buff.String())
	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	return txn.Commit()
}

func formatOrder(order string) string {
	value := order
	desc := ""
	if order[:1] == "-" {
		desc = " DESC"
		value = order[1:len(order)]
	}
	return fmt.Sprintf("%s%s", strings.TrimPrefix(value, "fields."), desc)
}

func formatFilter(filters url.Values) []*PQQueryFilter {
	clauses := make([]*PQQueryFilter, 0)
	for key, values := range filters {
		fmt.Println(key, values)
		f := key
		c := "+"
		comparerMatch := comparerRegex.FindStringSubmatch(f)
		if len(comparerMatch) > 0 {
			c = comparerMatch[1]
			f = strings.Replace(f, fmt.Sprintf("[%s]", c), "", 1)
		}
		if strings.HasPrefix(f, "sys.") {
			f = strings.Replace(f, "sys.", "sys", 1)
		} else {
			f = strings.TrimPrefix(f, "fields.")
		}
		clauses = append(clauses, &PQQueryFilter{
			Field:    f,
			Comparer: c,
			Value:    strings.Join(values, ","),
		})
	}
	return clauses
}
