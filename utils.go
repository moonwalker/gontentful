package gontentful

import (
	"fmt"
	"strings"
)

func fmtLocale(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", "_"))
}

func fmtTableName(contentType string, locale string) string {
	return fmt.Sprintf("%s_%s", strings.ToLower(contentType), fmtLocale(locale))
}

func fmtTablePublishName(contentType string, locale string) string {
	return fmt.Sprintf("%s_%s__publish", strings.ToLower(contentType), fmtLocale(locale))
}

func getFieldColumns(types []*ContentType, contentType string) []string {
	fieldColumns := make([]string, 0)
	for _, t := range types {
		if t.Sys.ID == contentType {
			for _, f := range t.Fields {
				fieldColumns = append(fieldColumns, strings.ToLower(f.ID))
			}
		}
	}
	return fieldColumns
}
