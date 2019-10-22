package gontentful

import (
	"fmt"
	"regexp"
	"strings"
)

func fmtLocale(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", "_"))
}

func fmtTableName(contentType string, locale string) string {
	return fmt.Sprintf("%s__%s", strings.ToLower(contentType), fmtLocale(locale))
}

func fmtTablePublishName(contentType string, locale string) string {
	return fmt.Sprintf("%s__%s__publish", strings.ToLower(contentType), fmtLocale(locale))
}

func getFieldColumns(types []*ContentType, contentType string) ([]string, map[string]string) {
	fieldColumns := make([]string, 0)
	refColumns := make(map[string]string)
	for _, t := range types {
		if t.Sys.ID == contentType {
			for _, f := range t.Fields {
				if !f.Omitted {
					colName := toSnakeCase(f.ID)
					fieldColumns = append(fieldColumns, colName)
					if f.Items != nil {
						linkType := getFieldLinkType(f.Items.LinkType, f.Items.Validations)
						if linkType != "" {
							refColumns[colName] = linkType
						}
					}
				}
			}
		}
	}
	return fieldColumns, refColumns
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")
var snake = regexp.MustCompile(`([_ ]\w)`)

func toSnakeCase(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToLower(strings.Join(a, "_"))
}

func toCamelCase(s string) string {
	return snake.ReplaceAllStringFunc(s, func(w string) string {
		return strings.ToUpper(string(w[1]))
	})
}
