package gontentful

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

const queryTemplate = `
SELECT * FROM {{ $.SchemaName }}._run_query(
{{- if $.Args }}{{- range $idx, $arg := $.Args -}}{{ $arg }},{{- end -}}{{- end -}}
'{{ $.TableName }}','{{ $.Locale }}','{{ $.DefaultLocale }}',
{{- if $.Fields }}ARRAY[
{{- range $idx, $field := $.Fields -}}
{{- if $idx -}},{{- end -}}'{{ $field }}'
{{- end -}}
]{{- else }}NULL{{ end -}},
{{- if $.Filters }}ARRAY[
{{- range $idx, $filter := $.Filters -}}
{{- if $idx -}},{{- end -}}({{ $filter }})::{{ $.SchemaName }}._filter
{{- end -}}]
{{- else -}}NULL{{- end -}},
'{{- $.Order -}}',
{{- $.Skip -}},
{{- $.Limit -}},
{{- $.Include -}}
)
`

var (
	comparerRegex      = regexp.MustCompile(`[^[]+\[([^]]+)+]`)
	joinedContentRegex = regexp.MustCompile(`(?:fields.)?([^.]+)\.sys\.contentType\.sys\.id`)
	foreignKeyRegex    = regexp.MustCompile(`([^.]+)\.(?:fields.)?(.+)`)
)

const (
	LINK  = "Link"
	ARRAY = "Array"

	DEFAULT_INCLUDE = 3
	MAX_INCLUDE     = 10
)

type PQQueryJoin struct {
	TableName string
	Localized bool
	Columns   map[string]*PGSQLMeta
}

type PGQuery struct {
	SchemaName    string
	TableName     string
	Locale        string
	DefaultLocale string
	Fields        *[]string
	Filters       *[]string
	Args          *[]string
	Order         string
	Limit         int
	Skip          int
	Include       int
}

func ParsePGQuery(schemaName string, defaultLocale string, q url.Values) *PGQuery {
	contentType := q.Get("content_type")
	q.Del("content_type")

	locale := q.Get("locale")
	q.Del("locale")
	if locale == "" {
		locale = defaultLocale
	}

	include := 0
	includeQ := q.Get("include")
	q.Del("include")
	if len(includeQ) > 0 {
		include, _ = strconv.Atoi(includeQ)
	} else {
		include = DEFAULT_INCLUDE
	}

	skip := 0
	skipQ := q.Get("skip")
	q.Del("skip")
	if skipQ != "" {
		skip, _ = strconv.Atoi(skipQ)
	}

	limit := 0
	limitQ := q.Get("limit")
	q.Del("limit")
	if limitQ != "" {
		limit, _ = strconv.Atoi(limitQ)
	}

	var fields *[]string
	fieldsQ := q.Get("select")
	q.Del("select")
	if fieldsQ != "" {
		fs := strings.Split(fieldsQ, ",")
		fields = &fs
	}

	order := q.Get("order")
	q.Del("order")

	return NewPGQuery(schemaName, contentType, locale, defaultLocale, fields, q, order, skip, limit, include)
}
func NewPGQuery(schemaName string, tableName string, locale string, defaultLocale string, fields *[]string, filters url.Values, order string, skip int, limit int, include int) *PGQuery {

	incl := include
	if incl > MAX_INCLUDE {
		incl = MAX_INCLUDE
	}

	q := PGQuery{
		SchemaName:    schemaName,
		TableName:     toSnakeCase(tableName),
		Locale:        fmtLocale(locale),
		DefaultLocale: fmtLocale(defaultLocale),
		//Fields:        formatFields(fields), // query ignores the fields for now and returns eveything
		Order:   formatOrder(order, tableName, defaultLocale),
		Skip:    skip,
		Limit:   limit,
		Include: incl,
	}

	if tableName == "game" {
		marketCode := filters.Get("fields.marketCode")
		device := filters.Get("fields.device")
		if marketCode != "" && device != "" {
			q.Args = &[]string{
				fmt.Sprintf("'%s'", marketCode),
				fmt.Sprintf("'%s'", device),
			}
			filters.Del("fields.marketCode")
			filters.Del("fields.device")
		}
	}

	q.Filters = createFilters(filters)

	return &q
}

func createFilters(filters url.Values) *[]string {
	if len(filters) > 0 {
		filterFields := make([]string, 0)
		for key, values := range filters {
			f, c := getFilter(key)
			if f != "" {
				vals := ""
				for _, val := range values {
					for i, v := range strings.Split(val, ",") {
						if i > 0 {
							vals = vals + ","
						}
						vals = vals + fmt.Sprintf("'%s'", v)
					}
				}
				filterFields = append(filterFields, fmt.Sprintf("'%s','%s',ARRAY[%s]", f, c, vals))
			}
		}
		return &filterFields
	}
	return nil
}

func getFilter(key string) (string, string) {
	f := key
	c := ""

	comparerMatch := comparerRegex.FindStringSubmatch(f)
	if len(comparerMatch) > 0 {
		c = comparerMatch[1]
		f = strings.Replace(f, fmt.Sprintf("[%s]", c), "", 1)
	}

	f = formatField(f)
	colName := toSnakeCase(f)

	if strings.Contains(colName, ".") {
		// content.fields.name%5Bmatch%5D=jack&content.sys.contentType.sys.id=gameInfo
		fkeysMatch := foreignKeyRegex.FindStringSubmatch(colName)
		if len(fkeysMatch) > 0 {
			if strings.HasPrefix(fkeysMatch[2], "sys.") {
				// ignore sys fields
				return "", ""
			}
			colName = fmt.Sprintf("%s.%s", fkeysMatch[1], fkeysMatch[2])
		}
	}

	return colName, c
}

func formatFields(fields *[]string) *[]string {
	if fields != nil {
		fmtFields := make([]string, 0)
		for _, f := range *fields {
			fmt := formatField(f)
			if fmt != "" {
				fmtFields = append(fmtFields, fmt)
			}
		}
		return &fmtFields
	}
	return fields
}

func formatField(f string) string {
	if f == "sys.id" {
		return "sys_id"
	}
	return strings.TrimPrefix(strings.TrimPrefix(f, "fields."), "sys.")
}

func formatOrder(order string, tableName string, defaultLocale string) string {
	if order == "" {
		return order
	}
	orders := make([]string, 0)
	for _, o := range strings.Split(order, ",") {
		value := o
		desc := ""
		if o[:1] == "-" {
			desc = " DESC"
			value = o[1:len(o)]
		}
		var field string
		if value == "sys.id" {
			field = fmt.Sprintf("%s__%s.sys_id", toSnakeCase(tableName), defaultLocale)
		} else if strings.HasPrefix(value, "sys.") {
			field = strings.TrimPrefix(value, "sys.")
			field = fmt.Sprintf("%s__%s.%s", toSnakeCase(tableName), defaultLocale, toSnakeCase(field))
		} else {
			field = fmt.Sprintf("\"%s\"", strings.TrimPrefix(value, "fields."))
		}

		orders = append(orders, fmt.Sprintf("%s%s", field, desc))
	}

	return strings.Join(orders, ",")
}

func (s *PGQuery) Exec(databaseURL string) (int64, string, error) {
	db, _ := sql.Open("postgres", databaseURL)

	defer db.Close()

	tmpl, err := template.New("").Parse(queryTemplate)

	if err != nil {
		return 0, "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return 0, "", err
	}

	fmt.Println(buff.String())

	var count int64
	var items string
	res := db.QueryRow(buff.String())
	err = res.Scan(&count, &items)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "[]", nil
		}
		return 0, "", err
	}

	return count, items, nil
}
