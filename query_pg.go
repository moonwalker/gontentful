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
	"time"
)

const includeQueryFormat = `
SELECT table_name FROM %s._entries WHERE sys_id = '%s'
`

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
{{ if ne $.Locale $.DefaultLocale }}LEFT JOIN {{ $.SchemaName }}.{{ $.TableName }}_{{ $.Locale }} {{ $.TableName }}_{{ $.Locale }} ON {{ $.TableName }}_{{ $.DefaultLocale }}.sys_id = {{ $.TableName }}_{{ $.Locale }}.sys_id{{ end }}
{{ if $.Filters }}WHERE
{{- range $fidx, $filter := $.Filters }}
{{- if $fidx }} AND {{- end }}
{{ .Field }} {{ .Comparer }} '{{ .Value }}'
{{ end -}}
{{- end -}}
{{- if $.Order }}ORDER BY {{ $.Order }}{{ end -}}
{{- if $.Limit }}LIMIT {{ $.Limit }}{{ end -}}
{{- if $.Skip }}OFFSET {{ $.Skip }}{{ end -}}
`

var (
	comparerRegex = regexp.MustCompile(`.+\[(^\])+]=.+`)
)

const (
	LINK  = "Link"
	ARRAY = "Array"
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
	Order         string
	Filters       []*PQQueryFilter
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

	res, err := s.execute(db, 0)
	fmt.Println("Exec", res)
	return err
}

func (s *PGQuery) execute(db *sql.DB, includeLevel int) (map[string]interface{}, error) {
	// fmt.Println("executing meta query for", s.TableName)
	// fmt.Println(fmt.Sprintf(metaQueryFormat, s.SchemaName, s.TableName))
	metas, err := db.Query(fmt.Sprintf(metaQueryFormat, s.SchemaName, s.TableName))
	if err != nil {
		return nil, err
	}
	defer metas.Close()
	fields := make(map[string]struct{})
	allFields := true
	if s.Fields != nil && len(s.Fields) > 0 {
		allFields = false
		for _, f := range s.Fields {
			fields[f] = struct{}{}
		}
	}
	s.Columns = make([]*PGSQLMeta, 0)
	includedAssets := make([]int, 0)
	includedEntries := make([]int, 0)
	index := 0
	for metas.Next() {
		meta := PGSQLMeta{}
		err := metas.Scan(&meta.Name, &meta.Type, &meta.LinkType, &meta.Items, &meta.Localized)
		if err != nil {
			return nil, err
		}
		_, ok := fields[meta.Name]
		if allFields || ok {
			s.Columns = append(s.Columns, &meta)
			if includeLevel < s.Include {
				switch getMetaLinkType(meta) {
				case ENTRY:
					includedEntries = append(includedEntries, index)
					break
				case ASSET:
					includedAssets = append(includedAssets, index)
					break
				}
			}
		}
		index = index + 1
	}
	tmpl, err := template.New("").Parse(queryTemplate)
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return nil, err
	}

	fmt.Println(buff.String())
	res, err := db.Query(buff.String())
	if err != nil {
		return nil, err
	}
	defer res.Close()
	entry := make(map[string]interface{})
	for res.Next() {
		values := make([]interface{}, len(s.Columns))
		for i := range values {
			values[i] = new(sql.RawBytes)
		}
		err := res.Scan(values...)
		if err != nil {
			return nil, err
		}
		for i, c := range s.Columns {
			entry[c.Name] = convertToType(values[i].(*sql.RawBytes), c)
		}
		if includeLevel < s.Include {
			err = s.includeAll(db, entry, includedEntries, includedAssets, includeLevel)
		}
		if err != nil {
			return nil, err
		}
	}
	return entry, nil
}

func getMetaLinkType(meta PGSQLMeta) string {
	switch meta.Type {
	case LINK:
		return meta.LinkType
	case ARRAY:
		return meta.Items
	}
	return ""
}

func convertToType(bytes *sql.RawBytes, column *PGSQLMeta) interface{} {
	str := string(*bytes)
	if str == "" {
		return nil
	}
	switch column.Type {
	case "Integer":
		i, _ := strconv.ParseInt(string(*bytes), 10, 64)
		return i
	case "Number":
		f, _ := strconv.ParseFloat(string(*bytes), 64)
		return f
	case "Date":
		d, _ := time.Parse(time.RFC3339, string(*bytes))
		return d
	case "Boolean":
		b, _ := strconv.ParseBool(string(*bytes))
		return b
	case "Array":
		str = strings.TrimPrefix(str, "{")
		str = strings.TrimSuffix(str, "}")
		vals := strings.Split(str, ",")
		switch column.Items {
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

func formatFilter(filters url.Values) []*PQQueryFilter {
	clauses := make([]*PQQueryFilter, 0)
	for key, values := range filters {
		fmt.Println("formatFilter", key, values)
		f := key
		c := "="
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

func (s *PGQuery) includeAll(db *sql.DB, fields map[string]interface{}, includedEntries []int, includedAssets []int, includeLevel int) error {
	// for _, a := range includedAssets {
	// 	if fields[a] != nil {
	// 		txn
	// 	}
	// }
	// fmt.Println("includedEntries", fields)
	for _, idx := range includedEntries {
		col := s.Columns[idx]
		// fmt.Println(idx, s.Columns[idx])
		if fields[col.Name] != nil {
			switch col.Type {
			case ARRAY:
				f, err := s.getBySysIDs(db, fields[col.Name].([]string), includeLevel)
				if err != nil {
					return err
				}
				fields[col.Name] = f
				//fmt.Println("getBySysIDs", col.Name, f)
				break
			case LINK:
				str := fields[col.Name].(string)
				if str != "" {
					f, err := s.getBySysIDs(db, []string{str}, includeLevel)
					if err != nil {
						return err
					}
					fields[col.Name] = f[0]
					//fmt.Println("getBySysIDs", col.Name, f)
				}
				break
			}
		}
	}
	return nil
}

func (s *PGQuery) getBySysIDs(db *sql.DB, sys_ids []string, includeLevel int) ([]map[string]interface{}, error) {
	res := make([]map[string]interface{}, 0)
	for _, id := range sys_ids {
		filter := url.Values{}
		filter.Set("sys.id", id)
		tableName := ""
		// fmt.Println(fmt.Sprintf(includeQueryFormat, s.SchemaName, id))
		erow := db.QueryRow(fmt.Sprintf(includeQueryFormat, s.SchemaName, id))
		err := erow.Scan(&tableName)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Println("No rows were returned!", id)
				continue
			}
			return nil, err
		}
		q := NewPGQuery(s.SchemaName, tableName, s.Locale, s.DefaultLocale, nil, filter, "", 0, 1, s.Include)
		r, err := q.execute(db, includeLevel+1)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
