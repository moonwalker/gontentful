package gontentful

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/jmoiron/sqlx"
)

const queryTemplate = `
{{- if .SchemaName -}}
	SET search_path='{{ .SchemaName }}';
{{- end }}
SELECT * FROM {{ .TableName }}_query(
'{{ .Locale }}',
{{- if $.Filters }}ARRAY[
{{- range $idx, $filter := $.Filters -}}
{{- if $idx -}},{{- end -}}'{{ $filter }}'
{{- end -}}]
{{- else -}}NULL{{- end -}},
'{{- $.Order -}}',
{{- $.Skip -}},
{{- $.Limit -}}
)
`

var (
	comparerRegex      = regexp.MustCompile(`[^[]+\[([^]]+)+]`)
	joinedContentRegex = regexp.MustCompile(`(?:fields.)?([^.]+)\.sys\.contentType\.sys\.id`)
	foreignKeyRegex    = regexp.MustCompile(`([^.]+)\.(?:fields.)?(.+)`)
	once               = new(sync.Once)
	db                 *sqlx.DB
)

const (
	LINK  = "Link"
	ARRAY = "Array"
)

type PGQuery struct {
	SchemaName string
	TableName  string
	Locale     string
	Filters    *[]string
	Order      string
	Limit      int
	Skip       int
}

func ParsePGQuery(schemaName string, defaultLocale string, q url.Values) *PGQuery {
	contentType := q.Get("content_type")
	q.Del("content_type")

	locale := q.Get("locale")
	q.Del("locale")
	if locale == "" || locale == "*" {
		locale = defaultLocale
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

	order := q.Get("order")
	q.Del("order")

	q.Del("include")
	q.Del("select")

	return NewPGQuery(schemaName, contentType, locale, q, order, skip, limit)
}
func NewPGQuery(schemaName string, contentType string, locale string, filters url.Values, order string, skip int, limit int) *PGQuery {
	tableName := toSnakeCase(contentType)
	q := PGQuery{
		SchemaName: schemaName,
		TableName:  tableName,
		Locale:     fmtLocale(locale),
		Order:      formatOrder(order, tableName),
		Skip:       skip,
		Limit:      limit,
	}

	q.Filters = createFilters(filters)

	return &q
}

func createFilters(filters url.Values) *[]string {
	if filters != nil && len(filters) > 0 {
		filterFields := make([]string, 0)
		for key, values := range filters {
			vals := ""
			for _, val := range values {
				for i, v := range strings.Split(val, ",") {
					if i > 0 {
						vals = vals + ","
					}
					vals = vals + formatValue(v)
				}
			}
			f := getFilterFormat(key, vals, values)
			if f != "" {
				filterFields = append(filterFields, f)
			}
		}
		if len(filterFields) > 0 {
			return &filterFields
		}
	}
	return nil
}

func getFilterFormat(key string, value string, values []string) string {
	f := key
	c := ""

	comparerMatch := comparerRegex.FindStringSubmatch(f)
	if len(comparerMatch) > 0 {
		c = comparerMatch[1]
		f = strings.Replace(f, fmt.Sprintf("[%s]", c), "", 1)
	}

	f = formatField(f)
	if f == "" {
		return f
	}

	if strings.Contains(f, ".") {
		return ""
	}
	// if strings.Contains(colName, ".") {
	// 	// content.fields.name%5Bmatch%5D=jack&content.sys.contentType.sys.id=gameInfo
	// 	// content.sys.contentType.sys.id=gameId&deviceConfigurations.sys.id=1yyHAve4aE6AQgkIyYG4im
	// 	fkeysMatch := foreignKeyRegex.FindStringSubmatch(colName)
	// 	if len(fkeysMatch) > 0 {
	// 		if fkeysMatch[2] != "sys.id" && strings.HasPrefix(fkeysMatch[2], "sys.") {
	// 			// ignore sys fields
	// 			return "", ""
	// 		}
	// 		colName = fmt.Sprintf("%s.%s", fkeysMatch[1], fkeysMatch[2])
	// 	}
	// }

	col := toSnakeCase(f)
	switch c {
	case "":
		return fmt.Sprintf("%s = %s", col, value)
	case "ne":
		return fmt.Sprintf("%s IS DISTINCT FROM %s", col, value)
	case "exists":
		return fmt.Sprintf("%s IS NOT NULL", col)
	case "lt":
		return fmt.Sprintf("%s < %s", col, value)
	case "lte":
		return fmt.Sprintf("%s <= %s", col, value)
	case "gt":
		return fmt.Sprintf("%s > %s", col, value)
	case "gte":
		return fmt.Sprintf("%s >= %s", col, value)
	case "match":
		return fmt.Sprintf("%s ILIKE ''%%'' || ''%s'' || ''%%''", col, strings.ReplaceAll(strings.Join(values, ","), "'", "'''"))
	case "all":
		return fmt.Sprintf("%s @> ARRAY[%s]", col, value)
	case "in":
		// IF isArray THEN
		// RETURN 	' && ARRAY[' || fmtVal || ']';
		return fmt.Sprintf("%s = ANY(ARRAY[%s])", col, value)
	case "nin":
		// IF isArray THEN
		// 			RETURN 	' && ARRAY[' || fmtVal || '] = false';
		return fmt.Sprintf("%s != ALL(ARRAY[%s])", col, value)
	}
	return ""
}

func formatValue(s string) string {
	if s == "true" || s == "false" {
		return fmt.Sprintf("%s", s)
	}

	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return fmt.Sprintf("%f", f)
	}

	return fmt.Sprintf("''%s''", s)
}

func formatField(f string) string {
	if f == "sys.id" {
		return "_sys_id"
	}
	return strings.TrimPrefix(strings.TrimPrefix(f, "fields."), "sys.")
}

func formatOrder(order string, tableName string) string {
	if order == "" {
		return order
	}
	orders := make([]string, 0)
	for _, o := range strings.Split(order, ",") {
		value := o
		desc := ""
		if o[:1] == "-" {
			desc = " DESC"
			value = o[1:]
		}
		var field string
		if value == "sys.id" {
			field = fmt.Sprintf("%s._sys_id", tableName)
		} else if strings.HasPrefix(value, "sys.") {
			field = fmt.Sprintf("%s._%s", tableName, strings.TrimPrefix(toSnakeCase(value), "sys."))
		} else {
			field = fmt.Sprintf("%s.%s", tableName, strings.TrimPrefix(toSnakeCase(value), "fields."))
		}

		orders = append(orders, fmt.Sprintf("%s%s NULLS LAST", field, desc))
	}

	return strings.Join(orders, ",")
}

func (s *PGQuery) Exec(databaseURL string) (int64, string, error) {
	var dbErr error
	once.Do(func() {
		db, dbErr = sqlx.Connect("postgres", databaseURL)
		if db != nil {
			db.SetMaxOpenConns(50)
			db.SetMaxIdleConns(50)                 // The default is defaultMaxIdleConns (= 2)
			db.SetConnMaxLifetime(5 * time.Minute) // The default is 0 (connections reused forever)
		}
	})
	if dbErr != nil {
		once = new(sync.Once)
		return 0, "", dbErr
	}

	tmpl, err := template.New("").Parse(queryTemplate)
	if err != nil {
		return 0, "", err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return 0, "", err
	}
	// fmt.Println(buff.String())

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
