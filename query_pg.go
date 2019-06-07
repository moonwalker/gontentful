package gontentful

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"golang.org/x/sync/syncmap"
)

const assetQueryTemplate = `
SELECT
{{- $first := true -}}
{{- range $colName, $field := $.SelectedFields -}}
{{- if $first }}{{ $first = false }}{{ else }},{{ end }}
{{ if and .Localized (ne $.Locale $.DefaultLocale) }}
COALESCE(_asset__{{ $.Locale }}.{{ .Name }},_asset__{{ $.DefaultLocale }}.{{ .Name }}) as {{ .Name }}
{{- else -}}
_asset__{{ $.DefaultLocale }}.{{ .Name }} as {{ .Name }}
{{- end -}}
{{- end }}
FROM {{ $.SchemaName }}._asset__{{ $.DefaultLocale }}{{ $.Suffix }} _asset__{{ $.DefaultLocale }}
{{- if ne $.Locale $.DefaultLocale }}
LEFT JOIN {{ $.SchemaName }}._asset__{{ $.Locale }}{{ $.Suffix }} _asset__{{ $.Locale }} ON _asset__{{ $.DefaultLocale }}.sys_id = _asset__{{ $.Locale }}.sys_id
{{- end }}
WHERE _asset__{{ $.DefaultLocale }}.sys_id = ANY(ARRAY[{{ index $.Fields 0 }}])
LIMIT 1`

const includeQueryFormat = `
SELECT sys_id, table_name FROM %s._entries WHERE sys_id = ANY(ARRAY[%s])`

const metaQueryFormat = `
SELECT
	name,
	type,
	link_type,
	coalesce(
		case
	  		when items IS NULL then null
	  		else items ->> 'linkType'
		end,
	'') as items,
	is_localized
--	is_required,
--  is_disabled,
--	is_omitted,
--	validations
FROM %s.%s___meta`

const queryTemplate = `
SELECT
{{- $first := true -}}
{{- range $colName, $field := $.SelectedFields -}}
{{- if $first }}{{ $first = false }}{{ else }},{{ end }}
{{ if and .Localized (ne $.Locale $.DefaultLocale)  }}COALESCE({{ $.TableName }}_{{ $.Locale }}.{{ .Name }},{{ $.TableName }}_{{ $.DefaultLocale }}.{{ .Name }}) as {{ .Name }}
{{- else }}{{ $.TableName }}_{{ $.DefaultLocale }}.{{ .Name }} as {{ .Name }}
{{- end -}}
{{- end }}
FROM {{ $.SchemaName }}.{{ $.TableName }}_{{ $.DefaultLocale }}{{ $.Suffix }} {{ $.TableName }}_{{ $.DefaultLocale }}
{{ if ne $.Locale $.DefaultLocale }}LEFT JOIN {{ $.SchemaName }}.{{ $.TableName }}_{{ $.Locale }}{{ $.Suffix }} {{ $.TableName }}_{{ $.Locale }} ON {{ $.TableName }}_{{ $.DefaultLocale }}.sys_id = {{ $.TableName }}_{{ $.Locale }}.sys_id{{ end }}
{{- range $foreignKey, $join := $.Joins }}
LEFT JOIN {{ $.SchemaName }}.{{ $join.TableName }}_{{ $.DefaultLocale }}{{ $.Suffix }} {{ $join.TableName }}_{{ $.DefaultLocale }} ON {{ $.TableName }}_{{ $.DefaultLocale }}.{{ $foreignKey }} = {{ $join.TableName }}_{{ $.DefaultLocale }}.sys_id
{{ if ne $.Locale $.DefaultLocale }}LEFT JOIN {{ $.SchemaName }}.{{ $join.TableName }}_{{ $.Locale }}{{ $.Suffix }} {{ $join.TableName }}_{{ $.Locale }} ON {{ $.TableName }}_{{ $.Locale }}.{{ $foreignKey }} = {{ $join.TableName }}_{{ $.Locale }}.sys_id{{ end }}
{{- end }}
{{ if gt (len $.Filters) 0 }}WHERE
{{- range $fidx, $filter := $.GetFilters }}
{{ if $fidx }} AND {{- end }}{{ $filter }}{{ end -}}
{{- end }}
{{ if $.Order }}ORDER BY {{ $.TableName }}_{{ $.Locale }}.{{ $.Order }}{{- end }}
{{ if $.Limit }}LIMIT {{ $.Limit }}{{- end }}
{{ if $.Skip }}OFFSET {{ $.Skip }}{{- end }}
`

const totalQueryTemplate = `
SELECT COUNT(*)
FROM {{ $.SchemaName }}.{{ $.TableName }}_{{ $.DefaultLocale }}{{ $.Suffix }} {{ $.TableName }}_{{ $.DefaultLocale }}
{{ if ne $.Locale $.DefaultLocale }}LEFT JOIN {{ $.SchemaName }}.{{ $.TableName }}_{{ $.Locale }}{{ $.Suffix }} {{ $.TableName }}_{{ $.Locale }} ON {{ $.TableName }}_{{ $.DefaultLocale }}.sys_id = {{ $.TableName }}_{{ $.Locale }}.sys_id{{ end }}
{{- range $foreignKey, $join := $.Joins }}
LEFT JOIN {{ $.SchemaName }}.{{ $join.TableName }}_{{ $.DefaultLocale }}{{ $.Suffix }} {{ $join.TableName }}_{{ $.DefaultLocale }} ON {{ $.TableName }}_{{ $.DefaultLocale }}.{{ $foreignKey }} = {{ $join.TableName }}_{{ $.DefaultLocale }}.sys_id
{{ if ne $.Locale $.DefaultLocale }}LEFT JOIN {{ $.SchemaName }}.{{ $join.TableName }}_{{ $.Locale }}{{ $.Suffix }} {{ $join.TableName }}_{{ $.Locale }} ON {{ $.TableName }}_{{ $.Locale }}.{{ $foreignKey }} = {{ $join.TableName }}_{{ $.Locale }}.sys_id{{ end }}
{{- end }}
{{ if gt (len $.Filters) 0 }}WHERE
{{- range $fidx, $filter := $.GetFilters }}
{{ if $fidx }} AND {{- end }}{{ $filter }}{{ end -}}
{{- end }}
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
	SchemaName     string
	TableName      string
	Locale         string
	DefaultLocale  string
	Fields         []string
	Filters        url.Values
	Order          string
	Limit          int
	Skip           int
	Include        int
	Joins          map[string]*PQQueryJoin
	Columns        map[string]*PGSQLMeta
	SelectedFields []*PGSQLMeta
	Suffix         string
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
	}
	if include == 0 {
		include = DEFAULT_INCLUDE
	}
	if include > MAX_INCLUDE {
		include = MAX_INCLUDE
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

	var fields []string
	fieldsQ := q.Get("select")
	q.Del("select")
	if fieldsQ != "" {
		fields = strings.Split(fieldsQ, ",")
	}

	order := q.Get("order")
	q.Del("order")

	return NewPGQuery(schemaName, contentType, locale, defaultLocale, fields, q, order, skip, limit, include, usePreview)
}
func NewPGQuery(schemaName string, tableName string, locale string, defaultLocale string, fields []string, filters url.Values, order string, skip int, limit int, include int, usePreview bool) *PGQuery {
	suffix := ""
	if !usePreview {
		suffix = "__publish"
	}
	return &PGQuery{
		SchemaName:    schemaName,
		TableName:     toSnakeCase(tableName),
		Locale:        fmtLocale(locale),
		DefaultLocale: fmtLocale(defaultLocale),
		Fields:        fields,
		Filters:       filters,
		Order:         formatOrder(order),
		Skip:          skip,
		Limit:         limit,
		Include:       include,
		Suffix:        suffix,
	}
}

var sb strings.Builder

func logQuery(query string) {
	sb.WriteString(query)
	sb.WriteString("\n")
}

func (s *PGQuery) Exec(databaseURL string) ([]map[string]interface{}, int64, error) {
	db, _ := sql.Open("postgres", databaseURL)

	items, total, err := s.execute(db, 0, true)
	d1 := []byte(sb.String())
	ioutil.WriteFile("/tmp/exec", d1, 0644)
	return items, total, err
}

func (s *PGQuery) execute(db *sql.DB, includeLevel int, withTotal bool) ([]map[string]interface{}, int64, error) {
	// fmt.Println("executing meta query for", s.TableName)
	logQuery(fmt.Sprintf(metaQueryFormat, s.SchemaName, s.TableName))
	metas, err := db.Query(fmt.Sprintf(metaQueryFormat, s.SchemaName, s.TableName))
	if err != nil {
		return nil, 0, err
	}
	defer metas.Close()
	fields := make(map[string]struct{})
	allFields := true
	if s.Fields != nil && len(s.Fields) > 0 {
		allFields = false
		for _, f := range s.Fields {
			fields[fieldToColumn(f)] = struct{}{}
		}
	}
	s.SelectedFields = make([]*PGSQLMeta, 0)
	s.Columns = make(map[string]*PGSQLMeta)
	includedAssets := make(map[string]struct{})
	includedEntries := make(map[string]struct{})
	for metas.Next() {
		meta := PGSQLMeta{}
		err := metas.Scan(&meta.Name, &meta.Type, &meta.LinkType, &meta.ItemsType, &meta.Localized)
		if err != nil {
			return nil, 0, err
		}
		_, ok := fields[meta.Name]
		if allFields || ok {
			s.Columns[meta.Name] = &meta
			s.SelectedFields = append(s.SelectedFields, &meta)
			if includeLevel < s.Include {
				switch getMetaLinkType(meta) {
				case ENTRY:
					includedEntries[meta.Name] = struct{}{}
					break
				case ASSET:
					includedAssets[meta.Name] = struct{}{}
					break
				}
			}
		}
	}

	err = s.getJoins(db)
	if err != nil {
		return nil, 0, err
	}
	var wg sync.WaitGroup
	var items []map[string]interface{}
	var ierr error
	var total int64
	var cerr error

	wg.Add(1)
	if withTotal {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tmpl, err := template.New("").Parse(totalQueryTemplate)
			if err != nil {
				cerr = err
				return
			}

			var buff bytes.Buffer
			err = tmpl.Execute(&buff, s)
			if err != nil {
				cerr = err
				return
			}

			logQuery(buff.String())
			res := db.QueryRow(buff.String())
			if err != nil {
				cerr = err
				return
			}
			count := 0
			err = res.Scan(&count)
			if err != nil {
				cerr = err
				return
			}
			total = int64(count)
		}()
	}

	go func() {
		defer wg.Done()
		tmpl, err := template.New("").Parse(queryTemplate)
		if err != nil {
			ierr = err
			return
		}

		var buff bytes.Buffer
		err = tmpl.Execute(&buff, s)
		if err != nil {
			ierr = err
			return
		}

		logQuery(buff.String())
		res, err := db.Query(buff.String())
		if err != nil {
			ierr = err
			return
		}
		defer res.Close()
		items = make([]map[string]interface{}, 0)

		for res.Next() {
			values := make([]interface{}, len(s.SelectedFields))
			for i := range values {
				values[i] = new(sql.RawBytes)
			}
			err := res.Scan(values...)
			if err != nil {
				ierr = err
				return
			}
			entry := syncmap.Map{}
			index := 0
			for _, c := range s.SelectedFields {
				bytes := values[index].(*sql.RawBytes)
				if bytes != nil {
					str := string(*bytes)
					if str != "" {
						entry.Store(toCamelCase(c.Name), convertToType(str, c))
					}
				}
				index = index + 1
			}
			if includeLevel < s.Include {
				err = s.includeAll(db, entry, includedEntries, includedAssets, includeLevel)
			}
			if err != nil {
				ierr = err
				return
			}
			item := make(map[string]interface{})
			entry.Range(func(key, value interface{}) bool {
				item[key.(string)] = value
				return true
			})
			items = append(items, item)
		}
	}()

	wg.Wait()

	if ierr != nil {
		return nil, 0, ierr
	}
	if cerr != nil {
		return nil, 0, cerr
	}
	return items, total, nil
}

func (s *PGQuery) getMetaColumns(db *sql.DB, tableName string) (map[string]*PGSQLMeta, error) {
	logQuery(fmt.Sprintf(metaQueryFormat, s.SchemaName, tableName))
	metas, err := db.Query(fmt.Sprintf(metaQueryFormat, s.SchemaName, tableName))
	if err != nil {
		return nil, err
	}
	defer metas.Close()
	columns := make(map[string]*PGSQLMeta)
	for metas.Next() {
		meta := PGSQLMeta{}
		err := metas.Scan(&meta.Name, &meta.Type, &meta.LinkType, &meta.ItemsType, &meta.Localized)
		if err != nil {
			return nil, err
		}
		columns[meta.Name] = &meta
	}
	return columns, nil
}

func fieldToColumn(field string) string {
	if strings.HasPrefix(field, "sys.") {
		return "sys_id"
	}
	return toSnakeCase(strings.TrimPrefix(field, "fields."))
}

func getMetaLinkType(meta PGSQLMeta) string {
	switch meta.Type {
	case LINK:
		return meta.LinkType
	case ARRAY:
		return meta.ItemsType
	}
	return ""
}

func convertToType(str string, column *PGSQLMeta) interface{} {
	if str == "" {
		return nil
	}
	switch column.Type {
	case "Integer":
		i, _ := strconv.ParseInt(str, 10, 64)
		return i
	case "Number":
		f, _ := strconv.ParseFloat(str, 64)
		return f
	case "Date":
		d, _ := time.Parse(time.RFC3339, str)
		return d
	case "Boolean":
		b, _ := strconv.ParseBool(str)
		return b
	case "Array":
		str = strings.TrimPrefix(str, "{")
		str = strings.TrimSuffix(str, "}")
		vals := strings.Split(str, ",")
		switch column.ItemsType {
		case "Integer":
			res := make([]int64, 0)
			for _, v := range vals {
				i, err := strconv.ParseInt(v, 10, 64)
				if err == nil {
					res = append(res, i)
				}
			}
			return res
		case "Number":
			res := make([]float64, 0)
			for _, v := range vals {
				f, err := strconv.ParseFloat(v, 64)
				if err == nil {
					res = append(res, f)
				}
			}
			return res
		case "Date":
			res := make([]time.Time, 0)
			for _, v := range vals {
				d, err := time.Parse(time.RFC3339, v)
				if err == nil {
					res = append(res, d)
				}
			}
			return res
		case "Boolean":
			res := make([]bool, 0)
			for _, v := range vals {
				b, err := strconv.ParseBool(v)
				if err == nil {
					res = append(res, b)
				}
			}
			return res
		}
		return vals
	}
	return str
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

func (s *PGQuery) getJoins(db *sql.DB) error {
	s.Joins = make(map[string]*PQQueryJoin)
	for key, values := range s.Filters {
		// fields.content.fields.name%5Bmatch%5D=jack&fields.content.sys.contentType.sys.id=gameInfo
		joinedContentMatch := joinedContentRegex.FindStringSubmatch(key)
		if len(joinedContentMatch) > 0 {
			s.Filters.Del(key)

			join := &PQQueryJoin{
				TableName: fieldToColumn(values[0]),
			}

			metas, err := s.getMetaColumns(db, join.TableName)
			if err != nil {
				return err
			}
			join.Columns = metas
			joinColumnName := fieldToColumn(joinedContentMatch[1])
			if s.Columns[joinColumnName] != nil {
				join.Localized = s.Columns[joinColumnName].Localized
				s.Joins[joinColumnName] = join
			}
		}
	}
	return nil
}

func (s *PGQuery) GetFilters() []string {
	clauses := make([]string, 0)
	for key, values := range s.Filters {
		f := key
		c := "="
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
				fkeyField := fkeysMatch[1]
				column := fkeysMatch[2]
				if s.Joins[fkeyField] != nil {
					meta := s.Joins[fkeyField].Columns[column]
					clauses = append(clauses, formatClause(meta, s.Joins[fkeyField].TableName, s.Locale, s.DefaultLocale, column, c, values))
				}
			}
		} else {
			meta := s.Columns[colName]
			clauses = append(clauses, formatClause(meta, s.TableName, s.Locale, s.DefaultLocale, colName, c, values))
		}
	}
	return clauses
}

func compareValuesString(c string, vals []string, column *PGSQLMeta) string {
	switch c {
	case "ne":
		return fmt.Sprintf(" <> %s", fmtValues(vals, column, ""))
	case "in":
		return fmt.Sprintf(" = ANY(%s)", fmtValues(vals, column, ""))
	case "nin":
		return fmt.Sprintf(" <> ANY(%s)", fmtValues(vals, column, ""))
	case "match":
		return fmt.Sprintf(" LIKE %s", fmtValues(vals, column, "%"))
	case "exists":
		return " <> NULL"
	case "lt":
		return fmt.Sprintf(" < %s", fmtValues(vals, column, ""))
	case "lte":
		return fmt.Sprintf(" <= %s", fmtValues(vals, column, ""))
	case "gt":
		return fmt.Sprintf(" > %s", fmtValues(vals, column, ""))
	case "gte":
		return fmt.Sprintf(" >=%s", fmtValues(vals, column, ""))
	}
	return fmt.Sprintf(" = %s", fmtValues(vals, column, ""))
}

func formatClause(column *PGSQLMeta, tableName string, locale string, defaultLocale string, field string, c string, vals []string) string {
	v := compareValuesString(c, vals, column)
	var sb strings.Builder
	sb.WriteString("(")
	if locale != defaultLocale && field != "sys_id" {
		sb.WriteString("(")
		sb.WriteString(tableName)
		sb.WriteString("_")
		sb.WriteString(locale)
		sb.WriteString(".")
		sb.WriteString(field)
		sb.WriteString(" IS NULL AND ")
		sb.WriteString(tableName)
		sb.WriteString("_")
		sb.WriteString(defaultLocale)
		sb.WriteString(".")
		sb.WriteString(field)
		sb.WriteString(v)
		sb.WriteString(") OR ")
	}
	sb.WriteString(tableName)
	sb.WriteString("_")
	if locale != defaultLocale && field != "sys_id" {
		sb.WriteString(locale)
	} else {
		sb.WriteString(defaultLocale)
	}

	sb.WriteString(".")
	sb.WriteString(field)
	sb.WriteString(v)
	sb.WriteString(")")
	return sb.String()
}

func fmtValues(values []string, meta *PGSQLMeta, prefix string) string {
	if meta == nil {
		// sys.id
		return fmt.Sprintf("'%s'", strings.Join(values, ","))
	}
	colType := meta.Type
	if colType == "Array" {
		colType = meta.ItemsType
	}
	vals := make([]string, 0)
	for _, v := range values {
		fv := v
		if colType == "Symbol" || colType == "Text" {
			fv = fmt.Sprintf("'%s%s%s'", prefix, fv, prefix)
		} else if colType == "Date" || colType == "Link" {
			fv = fmt.Sprintf("'%s'", fv)
		}
		vals = append(vals, fv)
	}
	res := strings.Join(vals, ",")
	if meta.Type == "Array" {
		res = fmt.Sprintf("ARRAY[%s]", res)
	}
	return res
}

func (s *PGQuery) includeAll(db *sql.DB, fields syncmap.Map, includedEntries map[string]struct{}, includedAssets map[string]struct{}, includeLevel int) error {
	var wg sync.WaitGroup
	var err error
	wg.Add(len(includedAssets))
	wg.Add(len(includedEntries))

	for a := range includedAssets {
		go func(col *PGSQLMeta) {
			defer wg.Done()
			colName := toCamelCase(col.Name)
			val, ok := fields.Load(colName)
			if ok {
				switch col.Type {
				case ARRAY:
					strs, ok := val.([]string)
					if ok && len(strs) > 0 {
						f, err := s.getAssetsByIDs(db, strs, col.Localized)
						if err != nil {
							return
						}
						fields.Store(colName, f)
					}
					break
				case LINK:
					str, ok := val.(string)
					if ok && str != "" {
						f, err := s.getAssetsByIDs(db, []string{str}, col.Localized)
						if err != nil {
							return
						}
						if len(f) > 0 {
							fields.Store(colName, f[0])
						}
					}
					break
				}
			}
		}(s.Columns[a])
	}

	for c := range includedEntries {
		go func(col *PGSQLMeta) {
			defer wg.Done()
			colName := toCamelCase(col.Name)
			val, ok := fields.Load(colName)
			if ok {
				switch col.Type {
				case ARRAY:
					strs, ok := val.([]string)
					if ok && len(strs) > 0 {
						f, err := s.getBySysIDs(db, strs, includeLevel)
						if err != nil {
							return
						}
						fields.Store(colName, f)
					}
					break
				case LINK:
					str, ok := val.(string)
					if ok && str != "" {
						f, err := s.getBySysIDs(db, []string{str}, includeLevel)
						if err != nil {
							return
						}
						if len(f) > 0 {
							fields.Store(colName, f[0])
						}
					}
					break
				}
			}
		}(s.Columns[c])
	}
	wg.Wait()

	return err
}

func (s *PGQuery) includeAssets(col *PGSQLMeta, db *sql.DB, fields map[string]interface{}) error {
	colName := toCamelCase(col.Name)
	if fields[colName] != nil {
		switch col.Type {
		case ARRAY:
			strs := fields[colName].([]string)
			if len(strs) > 0 {
				f, err := s.getAssetsByIDs(db, strs, col.Localized)
				if err != nil {
					return err
				}
				fields[colName] = f
			}
			break
		case LINK:
			str := fields[colName].(string)
			if str != "" {
				f, err := s.getAssetsByIDs(db, []string{str}, col.Localized)
				if err != nil {
					return err
				}
				if len(f) > 0 {
					fields[colName] = f[0]
				}
			}
			break
		}
	}
	return nil
}

func (s *PGQuery) getAssetsByIDs(db *sql.DB, sysIds []string, localized bool) ([]map[string]interface{}, error) {
	columns := make([]*PGSQLMeta, 0)
	for _, col := range assetColumns {
		columns = append(columns, &PGSQLMeta{
			Name:      col,
			Localized: localized,
		})
	}
	q := &PGQuery{
		SchemaName:     s.SchemaName,
		Locale:         s.Locale,
		DefaultLocale:  s.DefaultLocale,
		Fields:         []string{sysIdsToString(sysIds)},
		SelectedFields: columns,
	}
	tmpl, err := template.New("").Parse(assetQueryTemplate)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, q)
	if err != nil {
		return nil, err
	}

	logQuery(buff.String())
	res, err := db.Query(buff.String())
	if err != nil {
		return nil, err
	}
	defer res.Close()
	assets := make([]map[string]interface{}, 0)

	for res.Next() {
		values := make([]interface{}, len(assetColumns))
		for i := range values {
			values[i] = new(sql.RawBytes)
		}
		err := res.Scan(values...)
		if err != nil {
			return nil, err
		}
		file := make(map[string]interface{})
		for i, c := range assetColumns {
			bytes := values[i].(*sql.RawBytes)
			if bytes != nil {
				str := string(*bytes)
				if str != "" {
					file[toCamelCase(c)] = str
				}
			}
		}
		if len(file) > 0 {
			asset := make(map[string]interface{})
			asset["file"] = file
			assets = append(assets, asset)
		}
	}

	return assets, nil
}

func (s *PGQuery) getBySysIDs(db *sql.DB, sysIds []string, includeLevel int) ([]map[string]interface{}, error) {
	values := sysIdsToString(sysIds)
	logQuery(fmt.Sprintf(includeQueryFormat, s.SchemaName, values))
	rows, err := db.Query(fmt.Sprintf(includeQueryFormat, s.SchemaName, values))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := make([]map[string]interface{}, 0)
	for rows.Next() {
		sysID := ""
		tableName := ""
		err := rows.Scan(&sysID, &tableName)
		if err != nil {
			return nil, err
		}
		filter := url.Values{}
		filter.Set("sys.id", sysID)
		q := NewPGQuery(s.SchemaName, tableName, s.Locale, s.DefaultLocale, nil, filter, "", 0, 1, s.Include, s.Suffix == "")
		r, _, err := q.execute(db, includeLevel+1, false)
		if err != nil {
			return nil, err
		}
		res = append(res, r[0])
	}
	return res, nil
}

func sysIdsToString(sysIds []string) string {
	values := ""
	for i, id := range sysIds {
		c := ","
		if i == 0 {
			c = ""
		}
		values = fmt.Sprintf("%s%s'%s'", values, c, id)
	}
	return values
}

// CREATE OR REPLACE FUNCTION content.__test(tableName text, locale text, defaultLocale text, filters text[], joins text[][])
// --RETURNS SETOF RECORD AS $$
// RETURNS SETOF content.market_en__publish AS $$
// DECLARE
//     qs text;
// 	meta record;
// 	firstFilter boolean := true;
// 	filter text;
// 	js text[];
// BEGIN
// 	qs := 'SELECT ' || tableName || '_' || defaultLocale || '._id _id, ' || tableName || '_' || defaultLocale || '.sys_id sys_id, ';
// 	FOR meta IN
// 		EXECUTE 'SELECT
// 		name,
// 		is_localized
//         FROM content.' || tableName || '___meta' LOOP
// 		IF meta.is_localized AND locale <> defaultLocale THEN
// 			qs := 'COALESCE(' || tableName || '_' || locale || '.' || meta.name || ',' || tableName || '_' || defaultLocale || '.' || meta.name || ')';
// 		ELSE
// 	    	qs := qs || tableName || '_' || defaultLocale || '.' || meta.name;
// 		END IF;
// 	qs := qs || ' as ' || meta.name || ', ';
//     END LOOP;

// 	qs := qs || tableName || '_' || defaultLocale || '.version, ' ||
// 		tableName || '_' || defaultLocale || '.published_at, ' ||
// 		tableName || '_' || defaultLocale || '.published_by ' ||
// 	 	'FROM content.' || tableName || '_' || defaultLocale || '__publish ' || tableName || '_' || defaultLocale;

// 	IF locale <> defaultLocale THEN
// 		qs := qs || ' LEFT JOIN content.' || tableName || '_' || locale || '__publish ' || tableName || '_' || locale ||
// 		' ON ' || tableName || '_' || defaultLocale || '.sys_id = ' || tableName || '_' || locale || '.sys_id';
// 	END IF;

// 	IF joins IS NOT NULL THEN
// 		FOREACH js SLICE 1 IN ARRAY joins LOOP
// 			qs := qs || ' LEFT JOIN content.' || js[2] || '_' || defaultLocale || '__publish ' || js[2] || '_' || defaultLocale  || ' ON ' || tableName || '_' || defaultLocale || '.' || js[1] || ' = ' || js[2] || '_' || defaultLocale || '.sys_id';
// 			IF locale <> defaultLocale THEN
// 				qs := qs || ' LEFT JOIN content.' || js[2] || '_' || locale || '__publish ' || js[2] || '_' || locale  || ' ON ' || tableName || '_' || locale || '.' || js[1] || ' = ' || js[2] || '_' || locale || '.sys_id';
// 			END IF;
// 		END LOOP;
// 	END IF;

// 	RAISE NOTICE '%', qs;

// 	IF filters IS NOT NULL THEN
// 		qs := qs || ' WHERE ';
// 		FOREACH filter IN ARRAY filters LOOP
// 			IF firstFilter THEN
// 				firstFilter:= false;
// 			ELSE
// 				qs := qs || ' AND ';
// 			END IF;
// 			qs := qs || filter;
// 		END LOOP;
// 	END IF;

// 	RAISE NOTICE '%', qs;

// 	RETURN QUERY EXECUTE qs;
// END;
// $$ LANGUAGE 'plpgsql';

// SELECT * FROM content.__test('market', 'sv', 'en', ARRAY['market_en.code = ''ROW'''], ARRAY[['locales', 'locale']]);
