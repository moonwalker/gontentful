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
SELECT * FROM {{ $.SchemaName }}._run_query('{{ $.TableName }}','{{ $.Locale }}','{{ $.DefaultLocale }}',
{{- if $.Fields }}ARRAY[
{{- range $idx, $field := $.Fields -}}
{{- if $idx -}},{{- end -}}'{{ $field }}'
{{- end -}}
]{{- else }}NULL{{ end -}},
{{- if $.Filters }}ARRAY[
{{- range $idx, $filter := $.Filters -}}
{{- if $idx -}},{{- end -}}'{{ $filter }}'
{{- end -}}],ARRAY[
{{- range $idx, $comparer := $.Comparers -}}
{{- if $idx -}},{{- end -}}'{{ $comparer }}'
{{- end -}}],ARRAY[
{{- range $idx, $value := $.FilterValues -}}
{{- if $idx -}},{{- end -}}'{{ $value }}'
{{- end -}}]
{{- else -}}NULL,NULL,NULL{{- end -}},
'{{- $.Order -}}',
{{- $.Skip -}},
{{- $.Limit -}},
{{- $.Include -}},
{{- if $.UsePreview }}true{{ else }}false{{ end -}}
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
	Comparers     *[]string
	FilterValues  *[]string
	Order         string
	Limit         int
	Skip          int
	Include       int
	UsePreview    bool
}

func ParsePGQuery(schemaName string, defaultLocale string, usePreview bool, q url.Values) *PGQuery {
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

	return NewPGQuery(schemaName, contentType, locale, defaultLocale, fields, q, order, skip, limit, include, usePreview)
}
func NewPGQuery(schemaName string, tableName string, locale string, defaultLocale string, fields *[]string, filters url.Values, order string, skip int, limit int, include int, usePreview bool) *PGQuery {
	filterFields, comparers, filterValues := createFilters(filters)
	incl := include
	if incl > MAX_INCLUDE {
		incl = MAX_INCLUDE
	}
	return &PGQuery{
		SchemaName:    schemaName,
		TableName:     toSnakeCase(tableName),
		Locale:        fmtLocale(locale),
		DefaultLocale: fmtLocale(defaultLocale),
		Fields:        fields,
		Filters:       filterFields,
		Comparers:     comparers,
		FilterValues:  filterValues,
		Order:         formatOrder(order),
		Skip:          skip,
		Limit:         limit,
		Include:       incl,
		UsePreview:    usePreview,
	}
}

func createFilters(filters url.Values) (*[]string, *[]string, *[]string) {
	if len(filters) > 0 {
		filterFields := make([]string, 0)
		comparers := make([]string, 0)
		filterValues := make([]string, 0)
		for key, values := range filters {
			f, c := getFilter(key)
			filterFields = append(filterFields, f)
			comparers = append(comparers, c)
			filterValues = append(filterValues, strings.Join(values, ","))
		}
		return &filterFields, &comparers, &filterValues
	}
	return nil, nil, nil
}

func getFilter(key string) (string, string) {
	f := key
	c := ""

	comparerMatch := comparerRegex.FindStringSubmatch(f)
	if len(comparerMatch) > 0 {
		c = comparerMatch[1]
		f = strings.Replace(f, fmt.Sprintf("[%s]", c), "", 1)
	}

	if strings.HasPrefix(f, "sys.") {
		f = strings.Replace(f, "sys.", "sys_", 1)
	} else {
		f = strings.TrimPrefix(f, "fields.")
	}

	colName := toSnakeCase(f)
	if strings.Contains(colName, ".") {
		// content.fields.name%5Bmatch%5D=cap
		fkeysMatch := foreignKeyRegex.FindStringSubmatch(f)
		if len(fkeysMatch) > 0 {
			colName = fmt.Sprintf("%s.%s", fkeysMatch[1], fkeysMatch[2])
		}
	}

	return colName, c
}

func formatOrder(order string) string {
	if order == "" {
		return order
	}
	value := order
	desc := ""
	if order[:1] == "-" {
		desc = " DESC"
		value = order[1:len(order)]
	}
	return fmt.Sprintf("%s%s", strings.TrimPrefix(value, "fields."), desc)
}

func (s *PGQuery) Exec(databaseURL string) (int64, string, error) {
	db, _ := sql.Open("postgres", databaseURL)

	tmpl, err := template.New("").Parse(queryTemplate)
	if err != nil {
		return 0, "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return 0, "", err
	}

	// d1 := []byte(sb.String())
	// ioutil.WriteFile("/tmp/exec", d1, 0644)

	//fmt.Println(buff.String())

	var count int64
	var items string
	res := db.QueryRow(buff.String())
	err = res.Scan(&count, &items)
	// fmt.Println(count, items)
	if err != nil {
		return 0, "", err
	}
	return count, items, nil
}
